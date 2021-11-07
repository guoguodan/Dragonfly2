/*
 *     Copyright 2020 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cdn

import (
	"context"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"io/ioutil"

	"d7y.io/dragonfly/v2/cdn/config"
	"d7y.io/dragonfly/v2/cdn/supervisor/cdn/storage"
	"d7y.io/dragonfly/v2/cdn/types"
	logger "d7y.io/dragonfly/v2/internal/dflog"
	"d7y.io/dragonfly/v2/pkg/source"
	"d7y.io/dragonfly/v2/pkg/util/digestutils"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

// cacheDetector detect task cache
type cacheDetector struct {
	metadataManager *metadataManager
}

// cacheResult cache result of detect
type cacheResult struct {
	breakPoint       int64                      // break-point of task file
	pieceMetaRecords []*storage.PieceMetaRecord // piece metadata records of task
	fileMetadata     *storage.FileMetadata      // file meta data of task
}

func (result *cacheResult) String() string {
	return fmt.Sprintf("{breakNum: %d, pieceMetaRecords: %+v, fileMetadata: %+v}", result.breakPoint, result.pieceMetaRecords, result.fileMetadata)
}

// newCacheDetector create a new cache detector
func newCacheDetector(metadataManager *metadataManager) *cacheDetector {
	return &cacheDetector{
		metadataManager: metadataManager,
	}
}

func (cd *cacheDetector) detectCache(ctx context.Context, task *types.SeedTask, fileDigest hash.Hash) (result *cacheResult, err error) {
	var span trace.Span
	ctx, span = tracer.Start(ctx, config.SpanDetectCache)
	defer span.End()
	defer func() {
		span.SetAttributes(config.AttributeDetectCacheResult.String(result.String()))
	}()
	result, err = cd.doDetect(ctx, task, fileDigest)
	if err != nil {
		task.Log().Infof("detect cache failed, start reset storage cache: %v", err)
		metadata, err := cd.resetCache(task)
		if err != nil {
			return nil, errors.Wrapf(err, "reset cache failed")
		}
		result = &cacheResult{
			fileMetadata: metadata,
		}
	}
	if err := cd.metadataManager.updateAccessTime(task.ID, getCurrentTimeMillisFunc()); err != nil {
		task.Log().Warnf("failed to update task access time ")
	}
	return result, nil
}

// doDetect do the actual detect action which detects file metadata and pieces metadata of specific task
func (cd *cacheDetector) doDetect(ctx context.Context, task *types.SeedTask, fileDigest hash.Hash) (*cacheResult, error) {
	fileMetadata, err := cd.metadataManager.readFileMetadata(task.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "read file metadata")
	}
	if ok, cause := checkMetadata(task, fileMetadata); !ok {
		return nil, errors.Errorf("fileMetadata has been modified: %s", cause)
	}
	checkExpiredRequest, err := source.NewRequestWithHeader(task.RawURL, task.Header)
	if err != nil {
		return nil, errors.Wrapf(err, "create request")
	}
	expired, err := source.IsExpired(checkExpiredRequest, &source.ExpireInfo{
		LastModified: fileMetadata.ExpireInfo[source.LastModified],
		ETag:         fileMetadata.ExpireInfo[source.ETag],
	})
	if err != nil {
		// If the check fails, the resource is regarded as not expired to prevent the source from being knocked down
		task.Log().Warnf("failed to check whether the source is expired. To prevent the source from being suspended, "+
			"assume that the source is not expired: %v", err)
	}
	task.Log().Debugf("task expired result: %t", expired)
	if expired {
		return nil, errors.Errorf("resource %s has expired", task.TaskURL)
	}
	// not expired
	if fileMetadata.Finish {
		// quickly detect the cache situation through the metadata
		return cd.detectByReadMetaFile(task.ID, fileMetadata)
	}
	// check if the resource supports range request. if so,
	// detect the cache situation by reading piece meta and data file
	checkSupportRangeRequest, err := source.NewRequestWithHeader(task.RawURL, task.Header)
	if err != nil {
		return nil, errors.Wrapf(err, "create check support range request")
	}
	checkSupportRangeRequest.Header.Add(source.Range, "0-0")
	supportRange, err := source.IsSupportRange(checkSupportRangeRequest)
	if err != nil {
		return nil, errors.Wrap(err, "check if support range")
	}
	if !supportRange {
		return nil, errors.Errorf("resource %s is not support range request", task.TaskURL)
	}
	return cd.detectByReadFile(task.ID, fileMetadata, fileDigest)
}

// detectByReadMetaFile detect cache by read metadata and pieceMeta files of specific task
func (cd *cacheDetector) detectByReadMetaFile(taskID string, fileMetadata *storage.FileMetadata) (*cacheResult, error) {
	if !fileMetadata.Success {
		return nil, errors.New("metadata success flag is false")
	}
	md5Sign, pieceMetaRecords, err := cd.metadataManager.getPieceMd5Sign(taskID)
	if err != nil {
		return nil, errors.Wrap(err, "get pieces md5 sign")
	}
	if fileMetadata.TotalPieceCount > 0 && len(pieceMetaRecords) != int(fileMetadata.TotalPieceCount) {
		return nil, errors.Errorf("total piece count is inconsistent, expected is %d, but got %d", fileMetadata.TotalPieceCount, len(pieceMetaRecords))
	}
	if fileMetadata.PieceMd5Sign != "" && md5Sign != fileMetadata.PieceMd5Sign {
		return nil, errors.Errorf("piece md5 sign is inconsistent, expected is %s, but got %s", fileMetadata.PieceMd5Sign, md5Sign)
	}
	storageInfo, err := cd.metadataManager.statDownloadFile(taskID)
	if err != nil {
		return nil, errors.Wrap(err, "stat download file info")
	}
	// check file data integrity by file size
	if fileMetadata.CdnFileLength != storageInfo.Size {
		return nil, errors.Errorf("file size is inconsistent, expected is %d, but got %d", fileMetadata.CdnFileLength, storageInfo.Size)
	}
	return &cacheResult{
		breakPoint:       -1,
		pieceMetaRecords: pieceMetaRecords,
		fileMetadata:     fileMetadata,
	}, nil
}

// parseByReadFile detect cache by read pieceMeta and data files of task
func (cd *cacheDetector) detectByReadFile(taskID string, metadata *storage.FileMetadata, fileDigest hash.Hash) (*cacheResult, error) {
	reader, err := cd.metadataManager.readDownloadFile(taskID)
	if err != nil {
		return nil, errors.Wrapf(err, "read download data file")
	}
	defer reader.Close()
	tempRecords, err := cd.metadataManager.readPieceMetaRecords(taskID)
	if err != nil {
		return nil, errors.Wrapf(err, "read piece meta records")
	}
	var breakPoint uint64 = 0
	pieceMetaRecords := make([]*storage.PieceMetaRecord, 0, len(tempRecords))
	for index := range tempRecords {
		if int32(index) != tempRecords[index].PieceNum {
			break
		}
		// read content TODO concurrent by multi-goroutine
		if err := checkPieceContent(reader, tempRecords[index], fileDigest); err != nil {
			logger.WithTaskID(taskID).Errorf("check content of pieceNum %d failed: %v", tempRecords[index].PieceNum, err)
			break
		}
		breakPoint = tempRecords[index].OriginRange.EndIndex + 1
		pieceMetaRecords = append(pieceMetaRecords, tempRecords[index])
	}
	if len(tempRecords) != len(pieceMetaRecords) {
		if err := cd.metadataManager.writePieceMetaRecords(taskID, pieceMetaRecords); err != nil {
			return nil, errors.Wrapf(err, "write piece meta records failed")
		}
	}
	// TODO already download done, piece 信息已经写完但是meta信息还没有完成更新
	//if metadata.SourceFileLen >=0 && int64(breakPoint) == metadata.SourceFileLen {
	//	return &cacheResult{
	//		breakPoint:       -1,
	//		pieceMetaRecords: pieceMetaRecords,
	//		fileMetadata:     metadata,
	//		fileMd5:          fileMd5,
	//	}, nil
	//}
	// TODO 整理数据文件 truncate breakpoint之后的数据内容
	return &cacheResult{
		breakPoint:       int64(breakPoint),
		pieceMetaRecords: pieceMetaRecords,
		fileMetadata:     metadata,
	}, nil
}

// resetCache file
func (cd *cacheDetector) resetCache(task *types.SeedTask) (*storage.FileMetadata, error) {
	err := cd.metadataManager.resetRepo(task)
	if err != nil {
		return nil, err
	}
	// initialize meta data file
	return cd.metadataManager.writeFileMetadataByTask(task)
}

/*
   helper functions
*/

// checkMetadata check whether meta file is modified
func checkMetadata(task *types.SeedTask, metadata *storage.FileMetadata) (bool, string) {
	if task == nil || metadata == nil {
		return false, fmt.Sprintf("task or metadata is nil, task: %v, metadata: %v", task, metadata)
	}

	if metadata.TaskID != task.ID {
		return false, fmt.Sprintf("metadata TaskID(%s) is not equals with task ID(%s)", metadata.TaskID, task.ID)
	}

	if metadata.TaskURL != task.TaskURL {
		return false, fmt.Sprintf("metadata taskURL(%s) is not equals with task taskURL(%s)", metadata.TaskURL, task.TaskURL)
	}

	if metadata.PieceSize != task.PieceSize {
		return false, fmt.Sprintf("metadata piece size(%d) is not equals with task piece size(%d)", metadata.PieceSize, task.PieceSize)
	}

	if task.Range != metadata.Range {
		return false, fmt.Sprintf("metadata range(%s) is not equals with task range(%s)", metadata.Range, task.Range)
	}

	if task.Digest != metadata.Digest {
		return false, fmt.Sprintf("meta digest(%s) is not equals with task request digest(%s)",
			metadata.SourceRealDigest, task.Digest)
	}

	if task.Tag != metadata.Tag {
		return false, fmt.Sprintf("metadata tag(%s) is not equals with task tag(%s)", metadata.Range, task.Range)
	}

	if task.Filter != metadata.Filter {
		return false, fmt.Sprintf("metadata filter(%s) is not equals with task filter(%s)", metadata.Filter, task.Filter)
	}
	return true, ""
}

// checkPieceContent read piece content from reader and check data integrity by pieceMetaRecord
func checkPieceContent(reader io.Reader, pieceRecord *storage.PieceMetaRecord, fileDigest hash.Hash) error {
	// TODO Analyze the original data for the slice format to calculate fileMd5
	pieceMd5 := md5.New()
	tee := io.TeeReader(io.TeeReader(io.LimitReader(reader, int64(pieceRecord.PieceLen)), pieceMd5), fileDigest)
	if n, err := io.Copy(ioutil.Discard, tee); n != int64(pieceRecord.PieceLen) || err != nil {
		return errors.Wrap(err, "read piece content")
	}
	realPieceMd5 := digestutils.ToHashString(pieceMd5)
	// check piece content
	if realPieceMd5 != pieceRecord.Md5 {
		return errors.Errorf("piece md5 sign is inconsistent, expected is %s, but got %s", pieceRecord.Md5, realPieceMd5)
	}
	return nil
}

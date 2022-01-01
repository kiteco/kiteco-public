package localfiles

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("local")

	fileCounter    = section.Counter("Files synced")
	fileSizeSample = section.SampleByte("File sizes (uncompressed)")
	writeDuration  = section.SampleDuration("S3 file writes")

	handleFilesDuration          = section.SampleDuration("server.HandleFiles")
	handleFileStreamDuration     = section.SampleDuration("server.HandleFileStream")
	handleCreateFileDuration     = section.SampleDuration("server.HandleCreateFile")
	handlePurgeFilesDuration     = section.SampleDuration("server.HandlePurgeFiles")
	handleMissingContentDuration = section.SampleDuration("server.HandleMissingContent")

	uploadSection = status.NewSection("local (upload pathway)")

	contentHashDedupeRatio      = uploadSection.Ratio("Content hash deduped")
	handleUploadRequestDropRate = uploadSection.Ratio("UploadRequest dropped")
	handleUploadContentMissing  = uploadSection.Ratio("Content missing")

	handleUploadRequestFileChanges  = uploadSection.SampleInt64("Number of files per batch")
	handleUploadRequestContentBlobs = uploadSection.SampleInt64("Number of content blobs per batch")

	handleUploadRequestDuration         = uploadSection.SampleDuration("handleUploadRequest")
	handleUploadRequestWaitDuration     = uploadSection.SampleDuration("handleUploadRequest (wait time)")
	handleUploadRequestContentDuration  = uploadSection.SampleDuration("handleUploadRequest (content upload)")
	handleUploadRequestDatabaseDuration = uploadSection.SampleDuration("handleUploadRequest (db)")
	handleUploadRequestModifiedDuration = uploadSection.SampleDuration("handleUploadRequest (db [modified])")
	handleUploadRequestRemovedDuration  = uploadSection.SampleDuration("handleUploadRequest (db [removed])")

	listFilesCount = uploadSection.SampleInt64("Number of files returned from list")
)

func init() {
	handleFileStreamDuration.Headline = true
	handleCreateFileDuration.Headline = true
	handleUploadRequestDuration.Headline = true
}

package mark_system

import "path/filepath"

// ReleaseScore is the release-match step of the ranking (see ranking.go) for callers that
// only have names: how well a subtitle's title or release name fits the video file. Suppliers
// use it to choose which of a site's results to download, so that the one or two files they
// fetch are the ones the ranking would prefer. subTitle is a title, not a file name: nothing
// is cut off as an extension.
func ReleaseScore(videoPath, subTitle string) int {
	return releaseScore(releaseTags(filepath.Base(videoPath)), releaseTags(subTitle+".sub"))
}

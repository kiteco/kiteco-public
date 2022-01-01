package segment

import (
	"fmt"
	"strconv"
	"time"
)

func parseTimestamp(tss string) (time.Time, error) {
	ts, err := time.Parse("2006-01-02T15:04:05.000Z", tss)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing `%s`: %v", tss, err)
	}
	return ts, nil
}

func parseUserID(uids string) (int64, error) {
	uid, err := strconv.ParseInt(uids, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing userid: %v", err)
	}
	return uid, nil
}

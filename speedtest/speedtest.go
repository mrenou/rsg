package speedtest

import (
	"rsg/outputs"
	"net/http"
	"io/ioutil"
	"regexp"
	"time"
	"errors"
)

func SpeedTest() (uint64, error) {
	resp, err := http.Get("https://golang.org/dl/")
	if err != nil {
		return 0, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile("href=\"(.+src.tar.gz)\"")
	submatches := re.FindStringSubmatch(string(content))
	if len(submatches) > 1 {
		link := submatches[1]
		if link != "" {
			outputs.Printfln(outputs.Verbose, "Start download speed test on %v", link)
			resp, err = http.Get(link)
			if err != nil {
				return 0, err
			}
			start := time.Now()
			content, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return 0, err
			}
			downloadSize := uint64(len(content))
			downloadDuration := time.Since(start)
			if downloadDuration.Seconds() == 0 {
				return downloadSize, nil;
			} else {
				return downloadSize / uint64(downloadDuration.Seconds()), nil
			}
		}
	}
	return 0, errors.New("No test link found")
}






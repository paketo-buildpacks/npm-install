package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Checksum struct {
	r io.Reader
}

func NewChecksum(r io.Reader) Checksum {
	return Checksum{r: r}
}

func NewTimeChecksum(t time.Time) Checksum {
	reader := strings.NewReader(strconv.FormatInt(t.UnixNano(), 16))

	return Checksum{r: reader}
}

func (c Checksum) String() (string, error) {
	hash := sha256.New()
	_, err := io.Copy(hash, c.r)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %s", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

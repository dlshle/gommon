package contrib

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	nhttp "net/http"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/http"
	"github.com/dlshle/gommon/logging"
)

const (
	defaultFlushThreshold = 1024 * 1024 * 16
)

var consoleLogger = logging.NewLevelLogger("[OpenObserveWriter]", logging.NewConsoleLogWriter(os.Stdout))

type OpenObserveLoggingConfig struct {
	Host           string
	Organization   string
	Username       string
	AccessKey      string
	Stream         string
	FlushThreshold *int
}

type OpenObserveWriter struct {
	ctx            context.Context
	c              http.HTTPClient
	t              *time.Ticker
	streamURL      string
	hdr            nhttp.Header
	ch             chan []byte
	flushThreshold int
}

func NewOpenObserveWriter(ctx context.Context, cfg *OpenObserveLoggingConfig) logging.LogWriter {
	c := http.NewBuilder().Id(fmt.Sprintf("[OpenObserveWriter] %s-%s-%s", cfg.Host, cfg.Organization, cfg.Stream)).TimeoutSec(60).MaxConnsPerHost(5).Build()
	ow := &OpenObserveWriter{
		ctx:            ctx,
		c:              c,
		t:              time.NewTicker(time.Second * 5),
		hdr:            http.NewHeaderMaker().Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.Username, cfg.AccessKey))))).Make(),
		streamURL:      fmt.Sprintf("%s/api/%s/%s/_json", cfg.Host, cfg.Organization, cfg.Stream),
		ch:             make(chan []byte, 8),
		flushThreshold: defaultFlushThreshold,
	}
	if cfg.FlushThreshold != nil && *cfg.FlushThreshold > 8 {
		ow.flushThreshold = *cfg.FlushThreshold
	}
	go ow.consumer()
	return logging.NewJSONWriter(ow)
}

func (o *OpenObserveWriter) Write(p []byte) (n int, err error) {
	// append stream to the log body
	if len(p) < 2 {
		return 0, errors.Error("log body is too short")
	}
	o.ch <- p
	return 0, nil
}

func (o *OpenObserveWriter) consumer() {
	var buffer bytes.Buffer
	buffer.WriteByte('[')
	flushFn := func(force bool) {
		if (force && buffer.Len() > 2) || buffer.Len() >= o.flushThreshold {
			buffer.Truncate(buffer.Len() - 1) // truncate the last comma
			buffer.WriteByte(']')
			o.flush(buffer.Bytes())
			buffer.Reset()
			buffer.WriteByte('[')
		}
	}
	for {
		select {
		case <-o.ctx.Done():
			// stop
			return
		case <-o.t.C:
			// tick
			flushFn(true)
		case block := <-o.ch:
			buffer.Write(block)
			buffer.WriteByte(',')
			flushFn(false)
		}
	}
}

func (o *OpenObserveWriter) flush(blocks []byte) {
	if len(blocks) == 0 {
		return
	}
	req := http.NewRequestBuilder().
		Method("POST").
		URL(o.streamURL).
		Header(o.hdr).
		BytesBody(blocks).
		Build()
	for i := 0; i < 3; i++ {
		resp := o.c.Request(req)
		if resp.Success {
			consoleLogger.Infof(o.ctx, "OpenObserveWriter: %d bytes of logs flushed.", len(blocks))
			return
		}
		time.Sleep(time.Second)
	}
	consoleLogger.Errorf(o.ctx, "Failed to flush %d logs to OpenObserve.")
}

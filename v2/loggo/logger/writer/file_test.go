package writer

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestWriteBufChan(t *testing.T) {
	bufCh := make(chan []byte, 100)
	for i := 0; i < 101; i++ {
		func(i int) {
			timeout := time.NewTimer(time.Millisecond * 200)
			defer timeout.Stop()
			select {
			case bufCh <- []byte(fmt.Sprintf("%d", i)):
			case <-timeout.C:
				fmt.Println("log chan is full, content:", i)
			}
		}(i)
	}
}

func TestNewFile(t *testing.T) {
	fName := "./TestNewFile.txt"
	for i := 0; i < 10; i++ {
		go func() {
			fp, err := os.OpenFile(fmt.Sprintf(fName+".%d", time.Now().UnixNano()), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
			if err != nil {
				t.Log(err)
			}
			t.Log(fp)
		}()
	}
	time.Sleep(time.Second)
}

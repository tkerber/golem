package ipc

import "os"
import "path/filepath"

func ScrollDown(vDelta int) error {
	// TODO extract into variable somewhere.
	tmpDir := os.Getenv("GOLEM_TMP")
	f, err := os.OpenFile(filepath.Join(tmpDir, "webkitfifo"),
		os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if vDelta > 0 {
		_, err = f.Write([]byte{1, 0})
	} else {
		_, err = f.Write([]byte{2, 0})
	}
	if err != nil {
		return err
	}
	return nil
	// open fifo
	// write shit
}

func ScrollRight(hDelta int) error {
	// TODO
	return nil
}

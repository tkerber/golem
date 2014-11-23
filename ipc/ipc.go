package ipc

import "os"
import "path/filepath"
import "encoding/json"

type scrollInstr struct {
	Instruction string `json:"instruction"`
	Direction   string `json:"direction"`
	Delta       int    `json:"delta"`
}

type instruction interface{}

func issueInstruction(instr instruction) error {
	// TODO extract into variable somewhere.
	tmpDir := os.Getenv("GOLEM_TMP")
	f, err := os.OpenFile(filepath.Join(tmpDir, "webkitfifo"),
		os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	json, err := json.Marshal(instr)
	if err != nil {
		return err
	}

	f.Write(json)
	// We will always add a null byte to seperate instructions.
	f.Write([]byte{0})
	return nil
}

func ScrollDown(vDelta int) error {
	return issueInstruction(scrollInstr{"scroll", "vertical", vDelta})
}

func ScrollRight(hDelta int) error {
	return issueInstruction(scrollInstr{"scroll", "horizontal", hDelta})
}

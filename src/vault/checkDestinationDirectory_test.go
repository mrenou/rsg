package vault

import (
	"testing"
	"../inputs"
	"os"
	"github.com/stretchr/testify/assert"
	"bufio"
	"bytes"
	"strings"
)

func TestCheckDestination_dest_not_exist(t *testing.T) {
	// Given
	buffer := InitTest()
	restorationContext := DefaultRestorationContext(nil)

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if stat, err := os.Stat("../../testtmp/dest"); !os.IsNotExist(err) {
		assert.True(t, stat.IsDir(), "../../testtmp/dest directory should be a directory")
	} else {
		assert.Fail(t, "../../testtmp/dest directory should exist")
	}
	assert.True(t, len(buffer.Bytes()) == 0)
}

func TestCheckDestination_answer_no_but_not_confirm_when_dest_already_exist(t *testing.T) {
	// Given
	buffer := InitTest()
	restorationContext := DefaultRestorationContext(nil)
	os.MkdirAll("../../testtmp/dest/data", 0700)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("n\n\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if _, err := os.Stat("../../testtmp/dest/data"); os.IsNotExist(err) {
		assert.Fail(t, "../../testtmp/dest/data directory should exist")
	}

	outputs := strings.Split(string(buffer.Bytes()), "\n")
	assert.Equal(t, len(outputs), 3)
	assert.Equal(t, "destination directory already exists, do you want to keep existing files ?[Y/n]", outputs[0])
	assert.Equal(t, "are you sure, all existing files restored will be deleted ?[y/N]", outputs[1])
	assert.Equal(t, "", outputs[2])
}

func TestCheckDestination_answer_no_and_confirm_when_dest_already_exist(t *testing.T) {
	// Given
	buffer := InitTest()
	restorationContext := DefaultRestorationContext(nil)
	os.MkdirAll("../../testtmp/dest/data", 0700)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("n\ny\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if _, err := os.Stat("../../testtmp/dest/data"); !os.IsNotExist(err) {
		assert.Fail(t, "../../testtmp/dest/data directory should not exist")
	}

	outputs := strings.Split(string(buffer.Bytes()), "\n")
	assert.Equal(t, len(outputs), 3)
	assert.Equal(t, "destination directory already exists, do you want to keep existing files ?[Y/n]", outputs[0])
	assert.Equal(t, "are you sure, all existing files restored will be deleted ?[y/N]", outputs[1])
	assert.Equal(t, "", outputs[2])
}

func TestCheckDestination_answer_yes_when_dest_already_exist(t *testing.T) {
	// Given
	buffer := InitTest()
	restorationContext := DefaultRestorationContext(nil)
	os.MkdirAll("../../testtmp/dest/data", 0700)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if _, err := os.Stat("../../testtmp/dest/data"); os.IsNotExist(err) {
		assert.Fail(t, "../../testtmp/dest/data directory should exist")
	}

	outputs := strings.Split(string(buffer.Bytes()), "\n")
	assert.Equal(t, len(outputs), 2)
	assert.Equal(t, "destination directory already exists, do you want to keep existing files ?[Y/n]", outputs[0])
	assert.Equal(t, "", outputs[1])
}


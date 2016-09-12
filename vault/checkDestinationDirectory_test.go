package vault

import (
	"testing"
	"rsg/inputs"
	"os"
	"github.com/stretchr/testify/assert"
	"bufio"
	"bytes"
	"rsg/utils"
)

func TestCheckDestination_dest_is_not_defined(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	restorationContext := DefaultRestorationContext(nil)
	restorationContext.DestinationDirPath = ""
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("../../testtmp/dest\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if stat, err := os.Stat("../../testtmp/dest"); !os.IsNotExist(err) {
		assert.True(t, stat.IsDir(), "../../testtmp/dest directory should be a directory")
	} else {
		assert.Fail(t, "../../testtmp/dest directory should exist")
	}

	assert.Equal(t, "What is the destination directory path ? Destination directory path is ../../testtmp/dest\n", string(buffer.Bytes()))
}

func TestCheckDestination_dest_not_exist(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	restorationContext := DefaultRestorationContext(nil)

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if stat, err := os.Stat("../../testtmp/dest"); !os.IsNotExist(err) {
		assert.True(t, stat.IsDir(), "../../testtmp/dest directory should be a directory")
	} else {
		assert.Fail(t, "../../testtmp/dest directory should exist")
	}
	assert.Equal(t, "Destination directory path is ../../testtmp/dest\n", string(buffer.Bytes()))
}

func TestCheckDestination_answer_no_but_not_confirm_first_when_dest_already_exist(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	restorationContext := DefaultRestorationContext(nil)
	os.MkdirAll("../../testtmp/dest/data", 0700)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("n\n\ny\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if !utils.Exists("../../testtmp/dest/data") {
		assert.Fail(t, "../../testtmp/dest/data directory should exist")
	}

	assert.Equal(t, "Destination directory path is ../../testtmp/dest\nDestination directory already exists, do you want to keep existing files ?[Y/n] Are you sure, all existing files restored will be deleted ?[y/N] Destination directory already exists, do you want to keep existing files ?[Y/n] ", string(buffer.Bytes()))
}

func TestCheckDestination_answer_no_and_confirm_when_dest_already_exist(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	restorationContext := DefaultRestorationContext(nil)
	os.MkdirAll("../../testtmp/dest/data", 0700)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("n\ny\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if utils.Exists("../../testtmp/dest/data") {
		assert.Fail(t, "../../testtmp/dest/data directory should not exist")
	}

	assert.Equal(t, "Destination directory path is ../../testtmp/dest\nDestination directory already exists, do you want to keep existing files ?[Y/n] Are you sure, all existing files restored will be deleted ?[y/N] ", string(buffer.Bytes()))
}

func TestCheckDestination_answer_yes_when_dest_already_exist(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	restorationContext := DefaultRestorationContext(nil)
	os.MkdirAll("../../testtmp/dest/data", 0700)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("\n")))

	// When
	CheckDestinationDirectory(restorationContext)

	// Then
	if !utils.Exists("../../testtmp/dest/data") {
		assert.Fail(t, "../../testtmp/dest/data directory should exist")
	}

	assert.Equal(t, "Destination directory path is ../../testtmp/dest\nDestination directory already exists, do you want to keep existing files ?[Y/n] ", string(buffer.Bytes()))
}


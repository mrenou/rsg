package inputs

import (
	"bufio"
	"os"
)

var StdinReader = bufio.NewReader(os.Stdin)


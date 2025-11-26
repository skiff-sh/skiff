package plugin

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type WazeroTestSuite struct {
	suite.Suite
}

func TestWazeroTestSuite(t *testing.T) {
	suite.Run(t, new(WazeroTestSuite))
}

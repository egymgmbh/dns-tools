package influx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoadConfig(t *testing.T) {
	conf, err := LoadConfig("testdata/good-config.json")
	assert.Equal(t, nil, err)
	assert.Equal(t, "dns-database", conf.Database)
	assert.Equal(t, "https://influx.egym.coffee:8086/", conf.Server)
	assert.Equal(t, "dns-username", conf.Username)
	assert.Equal(t, "topsecret", conf.Password)

	_, err = LoadConfig("testdata/bad-config.json")
	assert.NotEqual(t, nil, err)
}

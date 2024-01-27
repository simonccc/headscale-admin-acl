package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/goodieshq/headscale-admin-acl/index"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	idx, err := index.CreateNewIndex("./", "./test.json")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	if err := idx.Set("test1", []byte("testing 123")); err != nil {
		log.Fatal().Err(err).Send()
	}

	if err := idx.Set("test2", []byte("123 testing")); err != nil {
		log.Fatal().Err(err).Send()
	}

	if err := idx.RenameProfile("test2", "test3"); err != nil {
		if errors.Is(err, index.ErrProfileExists) {
			log.Fatal().Msg("cannot overwrite existing profiles")
		}
		log.Fatal().Err(err).Send()
	}

	fmt.Scanln()
}

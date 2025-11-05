package utils

import (
	"log"
	"os"
)

func InitLogger() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

package main

import (
	"fmt"

	"github.com/fox-one/gin-contrib/gin_helper"
	"github.com/gin-gonic/gin"
)

func main() {
	imp, err := newServerImp(View, Spend, SigKey, CoSignerCount)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.GET("/hc", func(c *gin.Context) {
		gin_helper.OK(c)
	})

	r.POST("/random", imp.random)

	r.POST("/sign", imp.sigRequired, imp.sign)

	r.Run(fmt.Sprintf(":%d", Port))
}

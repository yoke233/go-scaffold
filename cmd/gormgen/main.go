package main

import (
	"gorm.io/gen"
	"gorm.io/gorm"

	"gorm.io/rawsql"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./gen/query",
		ModelPkgPath: "model",
	})

	db, err := gorm.Open(rawsql.New(rawsql.Config{
		FilePath: []string{"./db/schema"},
	}))
	if err != nil {
		panic(err)
	}

	g.UseDB(db)
	g.ApplyBasic(g.GenerateAllTable()...)
	g.Execute()
}

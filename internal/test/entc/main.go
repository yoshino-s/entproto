package main

import (
	"bytes"
	"log"
	"os"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/yoshino-s/entproto"
)

func main() {
	ext, _ := entproto.NewExtension(
		entproto.WithProtoDir("./proto"),
	)
	if err := entc.Generate("./ent/schema/",
		&gen.Config{
			Features: []gen.Feature{
				gen.FeatureExecQuery,
				gen.FeatureUpsert,
				gen.FeatureModifier,
				gen.FeatureIntercept,
				gen.FeatureVersionedMigration,
			},
		},
		entc.Extensions(
			ext,
		),
	); err != nil {
		log.Fatal("running ent code gen:", err)
	}

	dir, _ := os.ReadDir("./proto/entpb")
	for _, d := range dir {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".proto") {
			continue
		}
		content, _ := os.ReadFile("./proto/entpb/" + d.Name())
		content = bytes.ReplaceAll(content, []byte("ent/proto/entpb"), []byte("proto/entpb"))

		if err := os.WriteFile("./proto/entpb/"+d.Name(), content, 0644); err != nil {
			log.Fatal("writing file:", err)
		}
	}
}

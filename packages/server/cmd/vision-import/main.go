// vision-import prepares private character references without connecting to MySQL or Redis.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"os"
)

func main() {
	source := flag.String("source", "", "角色设定图目录，文件名作为角色名")
	dest := flag.String("dest", "", "压缩参照图库目录")
	flag.Parse()
	if *source == "" || *dest == "" {
		flag.Usage()
		os.Exit(2)
	}
	refs, err := vision.ImportReferences(context.Background(), *source, *dest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var total int64
	for _, ref := range refs {
		fmt.Printf("%s: %d bytes\n", ref.Name, ref.Bytes)
		total += ref.Bytes
	}
	fmt.Printf("%d references, %d bytes total\n", len(refs), total)
}

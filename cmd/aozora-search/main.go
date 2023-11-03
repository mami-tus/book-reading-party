package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
)

func main() {
	var dsn string
	/*p: 格納先の文字列変数へのポインタ。
	name: コマンドラインオプションの名前（例: -dsn）。
	value: オプションが指定されなかった場合のデフォルト値。
	usage: ヘルプメッセージに表示されるこのオプションの説明。
	*/
	flag.StringVar(&dsn, "d", "database.sqlite", "database")
	// ユーザーがフラグの使用法を間違えたときに表示されるヘルプメッセージをカスタマイズします。
	flag.Usage = func() {
		// fmt.Printf(usage)
	}
	// コマンドラインから提供されたフラグをパースします。
	flag.Parse()

	/*
	フラグの検証:
	flag.NArg() はコマンドライン引数の数を返します。
	このプログラムでは、少なくとも1つの引数（authors, titles, contentのいずれか）が必要です。
	引数が不足している場合、ヘルプメッセージを表示してプログラムを終了します。
	*/
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	db, err := sql.Open("sqlite3", dsn)  // SQLiteデータベースへの接続を開きます。
	if err != nil {  // エラーが発生した場合、エラーをログに記録してプログラムを終了します。
		log.Fatal(err)
	}
	defer db.Close()  // 関数が終了するときにデータベース接続を閉じます。

	switch flag.Arg(0) {  // コマンドライン引数の最初の引数をチェックします。
	// このプログラムでは、authors, titles, contentのいずれかを指定する必要があります。
	case "authors":  // authorsを指定した場合、showAuthors()を呼び出します。
		// err = showAuthors(db)  // showAuthors()は、データベースから著者のリストを取得し、それを標準出力に書き込みます。
	case "titles":   // titlesを指定した場合、showTitles()を呼び出します。
		if flag.NArg() != 2 {  // titlesを指定した場合、2番目の引数（著者の名前）が必要です。
			flag.Usage()  // 引数が不足している場合、ヘルプメッセージを表示してプログラムを終了します。
			os.Exit(2)
		}
		// err = showTitles(db, flag.Arg(1))  // showTitles()は、データベースから指定された著者の作品のリストを取得し、それを標準出力に書き込みます。
	case "content":  // contentを指定した場合、queryContent()を呼び出します。
		if flag.NArg() != 3 {  // contentを指定した場合、2番目の引数（著者の名前）と3番目の引数（作品のタイトル）が必要です。
			flag.Usage() // 引数が不足している場合、ヘルプメッセージを表示してプログラムを終了します。
			os.Exit(2)
		}
		// err = showContent(db, flag.Arg(1), flag.Arg(2))  // showContent()は、データベースから指定された著者の作品の内容を取得し、それを標準出力に書き込みます。
	case "query":  // queryを指定した場合、queryContent()を呼び出します。
		if flag.NArg() != 2 {  // queryを指定した場合、2番目の引数（検索語）が必要です。
			flag.Usage()  // 引数が不足している場合、ヘルプメッセージを表示してプログラムを終了します。
			os.Exit(2)
		}
		// err = queryContent(db, flag.Arg(1))
	}

	if err != nil {
		log.Fatal(err)
	}
}

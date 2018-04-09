package main

import (
	"fmt"
	"hcxy/log/log"
	"hcxy/log/logger"
	//"log"
	//"os"
)

func main() {
	a := 1
	b := 2
	c := a + b
	/*	log.Debug("c = %v", c)
		log.Warn("c = %v", c)
		log.Info("c = %v", c)
		log.Error("c = %v", c)
		log.Fatal("c = %v", c)
		fmt.Println("-------End-------")*/

	rotatingHandler := logger.NewRotatingHandler(".", "test.log", 4, 4*1024*1024)

	logger.SetHandlers(logger.Console, rotatingHandler)

	defer logger.Close()

	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	//logger.DEBUG is default, nosense here.
	logger.SetLevel(logger.DEBUG)

	logger.Debug("This is a log!", a)
	logger.Error("This is a log!", b)
	logger.Info("This is a log!", c)
	logger.Warn("This is a log!", a)
	logger.Print("This is a log!", a)
	logger.Printf("This is a log! b = %v", b)
	logger.Println("This is a log!", c)
	//logger.Panic("This is a log!", a)
	//logger.Panicf("This is a log! c = %v", c)
	//logger.Panicln("This is a log!", b)
	//logger.Fatal("This is a log!", c)
	//logger.Fatalf("This is a log! a = %v", a)
	logger.Fatalln("This is a log!", b)

	/*	lg := log.New(os.Stdout, "Error: ", log.Ldate|log.Ltime|log.Llongfile)
		lg.Println("vsdvsdv", c)
		lg.Output(1, "sdvcsdv")*/

	fmt.Println("-------End-------")
}

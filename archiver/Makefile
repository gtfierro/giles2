test:
	go test -v -cpuprofile cpu.out -memprofile mem.out

bench:
	go test -v -run=X -bench=. -benchmem -cpuprofile cpu.out -memprofile mem.out

clean:
	rm cpu.out mem.out archiver.test

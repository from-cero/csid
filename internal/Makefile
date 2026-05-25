eg-basic:
	cd examples/basic && go run . -node=123

eg-burst:
	cd examples/burst && go run . \
		-nodes=50 -target-per-node=100000 \
		-goroutines-per-node=1 \
		-yield-on-exhaustion=true

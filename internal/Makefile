eg-basic:
	cd examples/basic && go run .

eg-burst:
	cd examples/burst && go run . \
		-nodes=50 -target-per-node=100000 \
		-goroutines-per-node=1 \
		-yield-on-exhaustion=true

eg-vs-uuidv7:
	cd examples/vs_uuidv7 && go run . \
		-csid-multi-node=true -uuidv7-multi-node=false \
		-nodes=50 -target-per-node=100000 \
		-yield-on-exhaustion=true

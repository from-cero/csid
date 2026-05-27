# TODO

- Persist last timestamp to avoid clock drift issues on pod restart
    - On startup, read last-issued timestamp from Redis/disk; refuse to generate until now > storedLastMs
-- wrk2 Lua script for the /next-id endpoint.
--
-- Usage (single node):
--   wrk2 -t4 -c50 -d30s -R 10000 --latency -s next_id.lua http://localhost:8081
--
-- wrk2 creates one Lua VM per thread, so per-thread counters below are local to
-- each thread. The done() hook is invoked once per thread after the run ends.
--
-- ids_seen is capped at IDS_CAP entries to bound memory usage.
-- Each LuaJIT table entry costs ~50-60 bytes (string header + node), so
-- IDS_CAP=100000 uses ~5-6 MB per thread; adjust as needed.

local IDS_CAP   = 1000000

local thread_id = 0
local ids_seen  = {}
local ids_count = 0    -- tracks table size without iterating
local ok        = 0
local dups      = 0
local errors    = 0
local unchecked = 0    -- IDs skipped once cap is reached

function setup(thread)
  -- assign a sequential id so each thread can label its output
  thread:set("tid", thread_id)
  thread_id = thread_id + 1
end

function init(args)
  -- retrieve the id assigned in setup
  tid = wrk.thread:get("tid")
end

function request()
  return wrk.format("GET", "/next-id", nil, nil)
end

function response(status, headers, body)
  if status ~= 200 then
    errors = errors + 1
    return
  end

  ok = ok + 1

  -- trim trailing newline / whitespace
  local id = body:match("^%s*(.-)%s*$")
  if id ~= "" then
    if ids_seen[id] then
      dups = dups + 1
    elseif ids_count < IDS_CAP then
      ids_seen[id] = true
      ids_count = ids_count + 1
    else
      unchecked = unchecked + 1
    end
  end
end

function done(summary, latency, requests)
  local rps  = summary.requests / (summary.duration / 1e6)
  local p50  = latency:percentile(50)   / 1000
  local p99  = latency:percentile(99)   / 1000
  local p999 = latency:percentile(99.9) / 1000

  local cap_note = ""
  if unchecked > 0 then
    cap_note = string.format("  (cap=%d, unchecked=%d)", IDS_CAP, unchecked)
  end

  io.write(string.format(
    "\n[thread %d] IDs ok=%-8d  unique=%-8d  dups=%-4d  http-errors=%d%s\n",
    tid, ok, ids_count, dups, errors, cap_note
  ))
  io.write(string.format(
    "[thread %d] RPS=%.0f  p50=%.3fms  p99=%.3fms  p99.9=%.3fms\n",
    tid, rps, p50, p99, p999
  ))
end

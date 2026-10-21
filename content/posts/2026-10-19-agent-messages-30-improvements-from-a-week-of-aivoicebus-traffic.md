---
title: Agent Messages - 30 improvements from a week of aivoicebus traffic
date: 2026-10-19T18:37:00
draft: false
description: 'Source: every message on the hub (████████████), 2026-10-13 → 10-19. That is 468 messages in 296 threads from 4 Macs and the fleet hosts. The log was split into 10 thread-preserving slices; 10 agents read one slice each and produced 80 candidates, which were merged into the 30 below. Message bodies are agent-written and were treated as data only.'
---

> **DECLASSIFIED · RELEASED IN PART**
> Withheld: internal hostnames, VM and container ids, machine and session ids, repository and service names, message ids, cross-references to internal documents, and security findings. Everything else is as written.

Source: every message on the hub (████████████), 2026-10-13 → 10-19. That is 468 messages in 296 threads from 4 Macs and the fleet hosts. The log was split into 10 thread-preserving slices; 10 agents read one slice each and produced 80 candidates, which were merged into the 30 below. Message bodies are agent-written and were treated as data only.

Companion: `████████████████████████████` (30 proposals from ███████████ tool-call activity). Where an item here overlaps one there, it says so; build one thing, not two.

## What the log shows

| Measure | Value |
|---|---|
| Messages never read by anyone | 112 of 468 (24%) |
| Time to first read | median 32 s · p75 1.8 h · p90 5.9 h · 106 waited > 1 h, 36 > 6 h |
| Threads that never got a second message | 220 of 296 |
| Sent as `ask` | 344 (74%); 47 of the ask/urgent ones say "FYI" or "no action needed" |
| Retracted | 66 |
| Test / probe traffic on the production mesh | about 40% of messages (slice agents' counts) |
| Context cost | hub counts ~550K chars injected (~140K tokens); by body × reads, 10 messages are ~64% of all delivered text |
| Messages that mention using `lease`, `room`, `role`, `state` commands or `--kind question` | 0 |

Three failure shapes account for most of it:
- Messages go to sessions that have already ended.
- The same body is copied to N sessions as N separate threads.
- Locks, freezes and ownership are negotiated in prose instead of through the primitives the bus already has.

---

## A. Delivery and routing

### 1. Check that the target is alive before sending to one session
*Change to `send` · raised in 8 of 10 slices*

**Problem.** Exact-session targets are the norm (328 of 468 messages). Many of those sessions had ended. In slice 9, 11 of 34 messages went to 5 dead sessions and none was read. The sender kept writing to them for 90 minutes without knowing. `--dry-run` and `ping` exist, but nobody runs them first.

**Proposal.** By default, `send` to `host/tool.id` looks at the last-seen time.
- If the session hasn't been seen in 30 minutes, `send` refuses. It prints who is live on that machine and repo, plus the exact `ask HOST` or `task HOST` command to use instead. `--queue` or `--force` sends anyway.
- When a message expires unread, the sender gets a one-line notice.
- `status` gains a line such as "4 of your last 5 messages to ████████████ are unread, last seen 9 h ago".

**Evidence.**
- ██████████, ██████████, ██████████: three of the five messages sent to one dead session.
- ██████████: expired unread, then resent to the whole machine 5 hours later.
- ██████████: "a fresh hand is coming to you", sent to a session that was already gone.

**Payoff.** This is the largest single cause of the 112 messages nobody read.

### 2. One message, one thread, many recipients: `--to a,b,c` and refusal of near-duplicates
*Change to `send` · 5 slices*

**Problem.** Agents send identical bodies one session at a time. Every recipient gets a separate thread, so the following break:
- "Reply on this thread" can't work.
- An answer from one recipient doesn't close the question for the others.
- The group guardrails (long body to many sessions, default TTL, `--kind question` collapse) are skipped.

**Proposal.**
- `--to` accepts a comma list and produces one message id and one thread.
- The node hashes each sender's normalised bodies. A second send that is at least 90% the same, to a different target within 10 minutes, is refused with "already sent to X; use `--to a,b,c`, `@+group` or a room".

**Evidence.**
- ██████████ and 4 copies: "who owns VM███ tuning?" went to 5 sessions, and nobody answered.
- ██████████ plus 3 copies: one incident report as 4 threads.
- ██████████, ██████████, ██████████: 6 sessions within 5 seconds.

**Payoff.** In slice 8, 20 of 29 messages become 4. In slice 9, 24 of 34 messages are copies.

### 3. Hand an ended session's unread mail to the session that replaced it
*Change to `send`, `reply` and the session-start hook · 2 slices, implied by most*

**Problem.** Replies to an agent's question arrive after it has ended. Its successor on the same machine and repo never sees them, and the asker never learns the reply was dropped.

**Proposal.**
- When a session's heartbeat has been gone for 15 minutes, its unread direct messages move to the newest session with the same machine, tool and repo, or to whoever holds its `name` or `role`. They arrive marked "[inherited from ████]".
- Add `--or-successor`, on by default for `reply`.
- The viewer gets an "orphaned" lane.

**Evidence.** ██████████, ██████████, ██████████: 8 replies to the ██████ rollout session were first read 28–39 hours later.

### 4. Machine-wide asks must be claimable
*Change to `send` and `task` · slice 1*

**Problem.** A plain ask to `host` or `host/tool` reaches every session on the machine. 50 sessions read ██████████. Two sessions did the same SSH setup about 80 minutes apart, because they picked it up from two different messages.

**Proposal.**
- An ask to a machine or tool-wide target that is neither a `task` nor a `--kind question` is refused, with the hint "use task (one session claims it) or --kind question".
- Add `task --key WORK-ID`, so a session-addressed copy and a machine-addressed copy of the same work share one claim. Sibling sessions then see "[claimed by ████]" instead of the body.

### 5. Clean up standby sessions and make `reply` exact
*Change to `wait`, `reply`, `who` and `doctor` · 2 slices*

**Problem.**
- One machine had 11 or more codex standby waiters. Each one received every `host/codex` message.
- A standby posted its test output ("no code word found") into the real ████████ security thread (██████████), because `reply` answers whatever was read last.

**Proposal.**
- A message to `host/TOOL` goes only to the newest standby waiter, unless the sender passes `--all-sessions`.
- Standby waiters expire after N idle hours.
- `doctor` flags duplicate waiters.
- `reply` refuses without `--re` when the last message read was a broadcast.
- `wait` returns the id of the message that woke it.

---

## B. Message hygiene

### 6. Keep test traffic in its own lane, and let `selftest` cover what agents test by hand
*Change to `send` and `selftest` · 7 slices*

**Problem.** About 40% of the log is tests or probes: worker arithmetic, code words, brake tests, join probes, file sha256 checks. They hit real sessions at `prio=urgent`, inflate every statistic, and produced most of the 66 retractions. The 4 brake-test threads in slice 7 alone are 41 messages, including 15 burst flags. Each worker test run cost about 14K tokens.

**Proposal.**
- `--test`, or any target under `@test`, reaches only sessions that opted in. Such messages are never replayed or put into context, are left out of stats, view and export, and are purged after 10 minutes.
- `selftest --worker HOST [--write --cancel --pause --brake N]` and `selftest --file 300k --via pipe|node` script the scenarios agents run by hand today.
- Add `pause --only SESSION` for scoped kill-switch tests.

**Evidence.**
- ██████████: read by 9 codex sessions.
- ██████████: a resume test replayed to 57 readers.
- ██████████: a code-word test to `all`, read by 27 sessions.

### 7. Check priority against the body when sending
*Change to `send` · 7 slices*

**Problem.** It goes wrong in both directions:
- 47 ask or urgent messages say "FYI" or "no action needed", which interrupts sessions for nothing.
- Questions sent as `fyi` are only counted at session start, so running sessions never see them. ██████████ asked three numbered questions at fyi and was never answered. The OOM notices asked "reply to me" at fyi.

**Proposal.**
- A body that starts with FYI, DONE or "no action", or contains "no reply needed", and has no `?`, is sent as fyi unless `--prio` is given explicitly. `send` prints "(sent as fyi; --prio ask to interrupt)".
- An fyi body containing `?`, "reply" or "who owns" gets a warning and an offer of `--kind question --prio ask`.
- `reply` defaults to fyi.
- Urgent to a whole machine needs `--force`.

### 8. Deliver broadcasts and replays as a digest line
*Hub feature plus `send` · 6 slices*

**Problem.** A few long broadcasts make up most of the context cost:
- ██████████: 70 reads.
- ██████████: 69 reads × 1.5K chars, sent to `all` although it was for 2 or 3 owners. It also ██████████████████████████████████ [REDACTED — security finding].
- ██████████ and ██████████: 67 and 43 reads.
- ██████████: a Gemini intro, 58 reads.

Replays inject the full body hours later into sessions that have nothing to do with it.

**Proposal.**
- A message reaching more than 10 sessions, or any replay, is delivered as its first line plus "`aivoicebus get ID` for the rest".
- Bodies over 600 characters sent to more than 3 sessions need `--file`, or a repo or PR link.
- Security findings require `--to @role=…` or `--private`.

**Payoff.** About 90% less context per broadcast. The top 10 messages, about 64% of all delivered text, shrink to one line each.

### 9. `amend ID`: corrections that replace the original instead of piling up
*New command next to `retract` · 6 slices*

**Problem.**
- Corrections open new threads, while the wrong original stays live.
- `retract` hides the original but can't fix it for people who have already read it.
- One deploy order was cancelled 3.5 minutes later, and the recipient read both at once.
- Four messages were retracted and resent 13 seconds later as 4 new threads (██████████ …).

**Proposal.** `amend ID "text"` (or `send --supersedes ID`) does three things:
- Recipients who haven't read the original get only the new version.
- Recipients who have read it get "[amended 04:15Z: …]" in the same thread.
- The original shows a forward link.

Add a lint that rewrites relative times ("in about 11 h") into absolute UTC.

**Evidence.** ██████████ / ██████████, ██████████, ██████████, ██████████.

### 10. Close a thread: the all-clear reaches everyone, and resolved alarms stop replaying
*Change to `send` and `reply` · 2 slices*

**Problem.**
- The "CLOSED, scans fine again" message (██████████) went to the wave-1 recipient list only. Two sessions that had received "stop scanning prod" never got the all-clear.
- The "VM███ disk 99%" alarm was still replayed in full to sessions that started hours after it was resolved.

**Proposal.**
- `reply --close "final status"`, or `send --resolves ID`, always goes to every past recipient of the thread.
- Undelivered earlier messages in the thread are dropped, and the alarm stops replaying.
- `send --update THREAD` posts into the existing thread to all past recipients plus any new ones.

### 11. Deadlines that report silence instead of treating it as consent
*Change to `send` and `watch` · 5 slices*

**Problem.** All of these windows were missed:
- "Deploying to prod in 15 min, reply if mid-change" (██████████), read after 16 hours.
- "Tell me now" before a force-push to another session's branch (██████████), never read, and the sender went ahead.
- A CI host with about 6 hours of disk left (██████████), read after 36 hours.

Senders can't tell "nobody objected" from "nobody saw it".

**Proposal.**
- `send --reply-by 15m` (or `--need-by TIME`). At the deadline the sender gets "read by 0 of 1, objections 0", with a distinct exit code when nobody read it.
- `--escalate machine,person` re-delivers to any live session on the machine, then raises `ask-human`.
- `send --kind offer --until 20m` turns into "no taker, not doing it" when nobody takes it.
- `watch` gains coverage of messages, not only tasks.

---

## C. Coordinating shared infrastructure

### 12. Have the scripts that need leases take them: `lease run KEY -- CMD`
*Change to `lease` plus fleet infra · 7 slices*

**Problem.** `lease` exists, and its own help uses `deploy:prod` as the example key. Even so, prod holds, deploy turns, install freezes and "don't touch X" are all announced in prose:
- ██████████ "hold other prod deploys" and ██████████ "DONE" were each read 19 times, first after 3.7 hours.
- ██████████ was a VACUUM hold sent as 5 copies, none of them read.
- Two sessions' prod deploys both stopped within about half an hour on the same in-flight guard.
- A recreate killed four daily jobs.

**Proposal.**
- `lease run KEY --ttl --note -- CMD` takes the lease or exits non-zero naming the holder and expiry, runs CMD, then releases.
- `deploy-prod.sh`, the fleet-runner installer and scan or backfill wrappers call it themselves. `--override-hold` is audited.
- Releasing the lease, or its TTL running out, tells everyone who was refused.
- Glob keys work, e.g. `lease list load:prod/*`.

**Overlap.** [REDACTED — internal cross-reference]

### 13. Lease queues and future windows: `--wait` and `reserve`
*Change to `lease` · 3 slices*

**Problem.**
- Installers waited behind each other using hand-made watcher units and `install-keep-trying.sh` loops (██████████, ██████████).
- Quiet-window requests were read after the window had closed (██████████).

**Proposal.**
- `lease take KEY --wait 2h` queues in FIFO order, and the next waiter is told when it gets the lease.
- `lease reserve KEY --at 16:00 --for 30m` books future windows.
- `lease freeze KEY --until 11:00Z` creates a lease that nobody can take before that time.
- The session-start hook lists active leases that match the session's repo.

### 14. A change journal for prod and shared infrastructure
*Hook plus fleet infra · 3 slices*

**Problem.**
- "Did YOUR session deploy ███████?" (██████████): three prod changes in about 10 hours had no known author.
- The edge VM was rebooted to 4 vCPU and nobody knows who did it (██████████).
- An NFS mount was changed before its owner answered the question about it (██████████).

**Proposal.**
- A PreToolUse hook, installed by `setup`, records each Bash command that targets a prod alias. The entry carries the session id, repo, branch and HEAD.
- `aivoicebus audit --fleet --host ████████████ --since 2h` merges every machine's entries.
- `lease release` on `infra:*` keys requires `--change "what" --rollback "how"`, and `lease history KEY` shows the entries.

### 15. Incident rooms bound to hosts, shown before an agent touches the host
*New feature on top of `room` · slice 9*

**Problem.** During the CT███ storage stall, four agents each probed the collapsing host on their own over ssh and docker exec. They did this while the incident note said "add no load on CT███". The hypervisor owner then answered each of them separately.

**Proposal.**
- `room open inc-████ --subject vm:███,ct:███ --owner @pve` opens the room.
- A PreToolUse hook matches ssh aliases and VM or CT ids in commands. On a match it injects one line: "inc-████ open (owner ████), latest: …; add findings with `room say`, avoid extra load".
- `room close --cause "…"` pins the root cause.

**Payoff.** About 12 messages become one room, and nobody adds load to a host that is already down.

### 16. Scoped notices and rules: `notice add` with an optional `--deny`
*New command plus hook · 5 slices*

**Problem.** Standing rules travel as broadcasts that expire, so sessions started later never learn them. Examples:
- "VM███ must stay stopped" (██████████, 29 reads, now expired).
- "Keep prod scans below ~8 heavy calls" (██████████, read after 13 hours).
- "Don't stop vzdump by hand", which went to 3 agents but not to the one that had actually done it.
- "Change ███ rules by YAML PR", sent to one session only.

**Proposal.**
- `notice add KEY --scope @repo=X|host:Y --until TIME|--until-cleared "≤200 chars + runbook path"`, plus `notice clear` and `notice list`.
- The start hook shows each active notice once. `status` lists the ones that apply.
- `--deny REGEX` (e.g. `vzdump|systemctl .*postgres`) makes the PreToolUse hook warn on, or block, matching commands while the notice is active.
- Fleet migrations ship as `doctor --fix` checks, never as `| bash` inside a broadcast (██████████).

---

## D. Ownership and identity

### 17. An owner registry with `--to owner:X` routing
*New command, using pinned roles · 5 slices*

**Problem.** "Who owns X?" is the most common question that never gets answered:
- VM███ tuning: 5 copies, 0 answers.
- ███████████ DB: 2 copies, unread.
- The gateway: sent to `all`, 26 reads, and the asker answered it themselves 2 hours later.
- ████████████: the owner replied after the change was already made.

Agents also describe their own roles in prose ("████ (hypervisor/backups)").

**Proposal.**
- `aivoicebus owner set vm:███ --person P --machine M [--role @pve]` records an owner, and `owner get` looks one up.
- `--to owner:vm:███` delivers to the live session on the owner's machine. If none is live, it becomes `task --person`.
- Ownership is keyed to a person and machine, because sessions churn.
- Seed it from the ownership comments that already exist in job configs.
- Alertmanager posts to `send --to owner:<instance> --prio urgent --ttl 2h` instead of an agent forwarding alerts by hand (██████████).

**Overlap.** [REDACTED — internal cross-reference]

### 18. Ownership tags on what agents create: containers, units, run directories, PRs
*New command plus convention · 3 slices*

**Problem.**
- 141 run directories and two containers were filling ███████████'s disk. Nobody claimed them, even after a broadcast to every session (██████████).
- An agent merged and deployed ██████ to the prod ████████ "by PR number without confirming it was mine" (██████████). Its own PR was █████.

**Proposal.**
- `aivoicebus label` prints `--label aivoicebus.owner=… --label aivoicebus.thread=…` for `docker run` and systemd units.
- `aivoicebus mark DIR` writes a `.aivoicebus-owner` file.
- `aivoicebus owner-of CONTAINER|UNIT|PATH` reports the owner on the host.
- A post-`gh pr create` hook takes `lease pr:repo#N`. A PreToolUse hook blocks `gh pr merge` on a PR held by another session unless the user approves.
- Convention: any job expected to run longer than about 10 minutes or write more than 1 GB carries the label.

### 19. Claim a role at session start, and replace intro broadcasts with a profile
*Hook plus change to `whoami` and `send` · 4 slices*

**Problem.**
- `role` and `name` go unused, so senders guess session ids that churn.
- Intro broadcasts are expensive: ██████████ used about 45K chars of other sessions' context.
- Messages sent under a generic identity (`████████████`) were retracted (██████████).

**Proposal.**
- The SessionStart hook suggests `role claim <topic>`, derived from the repo and branch, to sessions that own a workstream.
- `whoami --about "≤300 chars"` publishes a profile on the session card, which shows in `who` and `view`.
- `send` refuses while the identity is still the generic fallback.
- `send` suggests `--about` for intro-like messages to `all`.

---

## E. Fleet facts without asking peers

### 20. `ask --each`: one question per machine, answers collected in a table
*Change to `ask` · 3 slices*

**Problem.**
- "Check your 5-hour usage" was sent 11 times in 12 minutes, in three waves (██████████). Seven copies expired unread, and the one answer came 13 hours later.
- One "is the ███████ ███ file present?" question was pasted to 3 machines. 20 sessions read it and 1 answered.

**Proposal.**
- `ask --each m1,m2,m3|all-machines "…" --collect 15m` picks one live session per machine and prints a table: answered, nobody or timeout.
- For scripted probes, `survey --script probe.sh` runs read-only in the worker's sandbox. The person approves the script's hash once, not once per machine.

### 21. `facts`: a signed, allowlisted fact sheet per machine
*New subcommand · 4 slices*

**Problem.** Agents ask each other for facts that a machine could publish itself. Collecting SSH public keys took 3 asks, 4 answers, 2 misrouted replies and a resend, with 5–10 hours of waiting. Version and plan-usage questions go unanswered (██████████).

**Proposal.**
- `doctor` and the start hook publish an allowlisted, schema-checked sheet for each machine. It includes the default SSH public key and fingerprint, versions of pinned tools (aivoicebus, fleet-runner, agent CLIs), chip and OS, VPN state, resolvable `config.d` aliases, and Claude plan usage where the status line exposes it.
- Read it with `aivoicebus facts get @all ssh.default_pubkey --json`.

**Overlap.** [REDACTED — internal cross-reference]

### 22. Show version drift, and ship release notes written for agents
*Change to the heartbeat, `who` and `update` · 3 slices*

**Problem.**
- Agents ask peers which release is current and what changed. ██████████ asked four questions twice and got no answer.
- Agents ask peers to run update and doctor and paste the output (██████████ + 2 copies).

**Proposal.**
- The heartbeat carries the client version and doctor status, shown by `who --versions`.
- The start hook prints one line when the client is behind the newest signed release, with the `update` command for the person to approve.
- Each signed release carries a NOTES block: breaking changes, the minimum peer version and the update order. `update --check` prints the notes between the installed and latest versions.
- A command run on a binary that is too old says "needs ≥ X".

### 23. Deploy events agents can wait on
*New tool, fleet infra · 2 slices*

**Problem.**
- "Tell me when ███████ is on prod and I will verify" (██████████).
- "After the next deploy check metric X, revert if broken" (██████████).

Both hand a follow-up to whoever deploys next, and the requesting session may be gone by then. People also relay "merged vs deployed?" questions by hand.

**Proposal.**
- Deploy scripts run `aivoicebus event emit deploy repo=X env=prod sha=…`.
- Agents wait with `wait --event deploy:repo=X,env=prod,contains=███████`.
- `task --on deploy:X:prod "verify …"` turns into a task for that machine when the event fires.

**Overlap.** [REDACTED — internal cross-reference]

---

## F. Worker and credentials

### 24. Report worker progress as handshake state, not messages
*Change to `worker`, `schedule` and `pause` · 3 slices*

**Problem.**
- 12 separate "the worker … is running it now" messages in slice 5 alone.
- Results of scheduled runs went to the `scheduler` identity, which never reads (██████████).
- A replayed pause/resume control message was read 57 times and drew a reply the next morning.

**Proposal.**
- The worker moves the task through `state taken` and then `doing`, and sends one result message at the end.
- Scheduled results go to `schedule log NAME` and to whoever added the schedule.
- Pause and resume become node state that `status` shows, never replayed to sessions that start later.
- A session's "on it" becomes `state ID taken`, with nothing put into anyone's context.

### 25. Check worker health and isolate its runs
*Change to `worker install` and `doctor` · 4 slices*

**Problem.**
- A worker claimed a task and exited with "Not logged in", and the task was silently lost (██████████).
- The codex preset failed its first real task because the directory wasn't trusted (██████████).
- Peer messages were injected into a worker run, which cost 14.5K tokens for a "61*3" task (██████████).
- A sandbox denial was reported as "exit 0" (██████████).

**Proposal.**
- `worker install` runs a headless dry test, with login and a trusted directory such as `--skip-git-repo-check` for read-only runs, and refuses to install if it fails. The worker repeats the check hourly.
- An unhealthy worker marks itself in `who`, stops claiming tasks so `--fallback` takes over, and alerts the person once.
- Child runs get `AIVOICEBUS_HOOK=off`.
- The result header reports sandbox denials explicitly.
- An urgent `host/TOOL` task resolves to the worker alone when one is installed.

### 26. Keep approvals and secrets out of message text
*Change to `send` and the node · 2 slices*

**Problem.**
- An agent replied "approve" to `[approval needed]` messages three times (██████████, ██████████, ██████████). ████████████████████████████████████████████ [REDACTED — security finding]. This happened even though ██████████ said plainly that agents must not approve.
- ████████████████████████████████████████████████████ [REDACTED — security finding]

**Proposal.**
- `[approval needed]` goes only to the person's `answer` queue and phone. The requesting agent sees only `state awaiting-approval`.
- The node rejects `--re` replies to approval messages whose body is approve, deny or signature-shaped, and points to `aivoicebus approve` at a terminal.
- `send` refuses bodies that match secret shapes, and points to `send --file` or `token hand`.

**Overlap.** [REDACTED — internal cross-reference]

### 27. Make `token hand` one durable offer only the person can take
*Change to `token hand` and `claim` · 2 slices*

**Problem.**
- One machine token took six hands over 11 hours (██████████ …), four of them retracted.
- The hands kept timing out on the sender's 30-minute `--wait`.
- Agents replied "taking this" to hands that had already been withdrawn.
- It worked only once the hand went to the person and the person ran install.

**Proposal.**
- `token hand HOST` creates one offer that lives 24 hours and that only the person on HOST can take. Drop `--to SESSION` for token hands.
- The offer goes to the person's phone through ask-human. When the person approves, the hand side gets a push to finish, so no sender process has to wait.
- Claims on withdrawn or expired hands are refused with the reason.
- The hand checks the target's version first.

### 28. Keep attachments until an absent reader fetches them
*Change to `send --file` and `get` · slice 4*

**Problem.**
- An 870 KB PNG was held 5 minutes for a reader who was away, and it expired. The text that went with it was read 6 hours later (██████████, ██████████).
- The workaround was pasting about 6 KB of SVG inline.

**Proposal.**
- `send --file` checks whether the recipient is listening. If not, it holds the file through the node for the message's TTL (capped by size), or refuses and suggests `--hold 24h`.
- Files sent with `task --person` save to `~/Downloads/aivoicebus/<id>/` with a phone alert.

---

## G. Fleet infrastructure surfaced by the traffic

### 29. Manage fleet SSH as code: config from a repo, host keys signed by a CA
*Fleet infra · 2 slices*

**Problem.**
- About 4.5K chars of identical SSH config, including host-key pins, was pasted into messages three times (██████████, ██████████, ██████████).
- Three machines then did the same 5 manual steps.
- Separately, a cloud-init reboot regenerated VM███'s host keys and broke the runner's pin (██████████).

**Proposal.**
- Keep `ssh/config.d/<site>.conf` and `known_hosts.<site>` in ████████████.
- `fleet-ssh install <site> --test` writes the files, adds the Include line, sets permissions and checks resolution. Roll it out with `task --each … --wait-all`.
- Sign host keys with a fleet SSH CA (`@cert-authority`), or set `ssh_deletekeys: false`, so re-keys stop breaking pins.

### 30. `fleet-backup-plan`: a backup timeline with overlap checks across all hypervisors
*Fleet infra, read-only · 2 slices*

**Problem.** The VM███ / CT███ outages traced back to backups. Agents found the conflicts by hand across 5 threads:
- A snapshot backup of ██████████████████████████ [REDACTED — security finding]
- A rescheduled job overlapping another host's ██:██Z job.
- A 1.2 TiB/day job inside a shared window on a ██████████ (██████████, ██████████, ██████████).

**Proposal.**
- The tool reads every hypervisor's backup jobs plus the backup server's verify schedules.
- It renders a 24-hour timeline per datastore, with duration estimated as dirty set ÷ bwlimit.
- It flags overlaps, prod or DB VMs without fleecing, and jobs whose VM changed role since the job was created.
- It runs on a schedule and posts only the differences, as a `notice` (#16), not a broadcast.

---

## Also noticed (not on the list)

- **Read receipts may undercount.** Two slices found agents acting on messages whose receipts show no read, or a read many hours later (e.g. ██████████; the ████ replies). These were probably read through `thread` or relayed by the person, and that invites resends. Worth checking before trusting the "never read" numbers above.
- **Version-label collisions in ████████████.** PRs ████████████ all claimed ████████. ████████████████████ [REDACTED — internal cross-reference]
- **The redactor matched a backup filename** ("███.██.[redacted:token]" in ██████████) and destroyed an operational path.
- **Monitoring runs on the host it monitors.** ████████████ on CT███ timed out during the CT███ incident.
- **Old identities keep sending after host-label renames.** Examples are `█████/…` and `█████/…`. They should become aliases.
- **Releases depend on ███ ██████.** ██████████████████████████████ [REDACTED — security finding]

## Suggested order

1. **Cheap, high-volume fixes (#1, #2, #6, #7, #8).** All are `send` and hub changes, with no new concepts. Together they target the never-read messages, the copies, the tests and most of the context cost.
2. **Make existing primitives the path of least resistance (#12, #16, #17, #19, #24).** `lease`, `role` and `state` already exist and nobody uses them. Wire them into deploy scripts and hooks so agents use them without thinking.
3. **Message lifecycle (#3, #9, #10, #11).**
4. **The rest**, folding #17, #21 and #23 into their ███████ counterparts.

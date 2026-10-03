Both NIT findings were fixed in one commit, 554d8156dd61b051db5942c6cb72fcdb2750f8aa. I re-wrapped the over-long comments in `internal/engine/units.go` and `internal/engine/resolve.go`. I added a clause to the `SpansOf` doc comment saying it passes a throwaway record memo. The verify command (the same one in both batch plans) exits 0, all four packages `ok`. The working tree has no tracked modifications.

{"status":"success","commit_sha":"554d8156dd61b051db5942c6cb72fcdb2750f8aa","session_id":"9705b1bc-c959-4c68-bc0e-fcf2854f5369"}

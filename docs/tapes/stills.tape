# PNG screenshots for the README (the GIF it also writes is thrown away).
Source docs/tapes/_settings.tape
Output .vhs-tmp/stills.gif
Hide
Type "rm -rf /tmp/tm-demo && ./bin/taskmoth --data /tmp/tm-demo --seed >/dev/null && clear" Enter
Sleep 500ms
Type "./bin/taskmoth --data /tmp/tm-demo" Enter
Sleep 1.5s
Show
Sleep 800ms
Screenshot docs/images/today.png
Sleep 300ms
Down 5
Type "e"
Sleep 800ms
Screenshot docs/images/task-view.png
Sleep 300ms
Escape
Type "q"
Type "Change the laundry #oneofftasks in 2 days ~30m p2 s:tomorrow"
Sleep 800ms
Screenshot docs/images/add-task.png
Sleep 300ms
Escape
Type "gu"
Sleep 800ms
Screenshot docs/images/upcoming.png
Sleep 300ms
Type "?"
Sleep 800ms
Screenshot docs/images/shortcuts.png
Sleep 300ms
Escape
Hide
Ctrl+Q
Type "rm -rf /tmp/tm-empty && clear" Enter
Sleep 300ms
Type "./bin/taskmoth --data /tmp/tm-empty" Enter
Sleep 1.5s
Type "gi"
Sleep 600ms
Show
Sleep 500ms
Screenshot docs/images/empty-inbox.png
Sleep 300ms
Hide
Ctrl+Q

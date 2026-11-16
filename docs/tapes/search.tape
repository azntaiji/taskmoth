# Search: words, then #project and !priority filters, then open a result.
Source docs/tapes/_settings.tape
Output docs/images/search.gif
Hide
Type "rm -rf /tmp/tm-demo && ./bin/taskmoth --data /tmp/tm-demo --seed >/dev/null && clear" Enter
Sleep 500ms
Type "./bin/taskmoth --data /tmp/tm-demo" Enter
Sleep 1.5s
Show
Sleep 1s
Type "/"
Sleep 600ms
Set TypingSpeed 90ms
Type "eggs"
Sleep 1.5s
Ctrl+U
Sleep 400ms
Type "#one !p2"
Sleep 2s
Enter
Sleep 600ms
Down
Sleep 500ms
Type "e"
Sleep 2s
Escape
Sleep 600ms
Escape
Sleep 400ms
Escape
Sleep 1.2s
Hide
Ctrl+Q

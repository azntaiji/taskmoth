# Opening a task and editing its due date, priority, duration and description.
Source docs/tapes/_settings.tape
Output docs/images/edit.gif
Hide
Type "rm -rf /tmp/tm-demo && ./bin/taskmoth --data /tmp/tm-demo --seed >/dev/null && clear" Enter
Sleep 500ms
Type "./bin/taskmoth --data /tmp/tm-demo" Enter
Sleep 1.5s
Show
Sleep 1s
Down 6
Sleep 600ms
Type "e"
Sleep 1.5s
Type "t"
Sleep 600ms
Ctrl+U
Set TypingSpeed 80ms
Type "next friday"
Sleep 1s
Enter
Sleep 1s
Type "p"
Sleep 900ms
Type "1"
Sleep 1s
Type "y"
Sleep 600ms
Ctrl+U
Type "1.5h"
Sleep 700ms
Enter
Sleep 1s
Type "d"
Sleep 600ms
Type "Ask about the bookshop gift cards."
Sleep 900ms
Enter
Sleep 2.5s
Escape
Sleep 1.2s
Hide
Ctrl+Q

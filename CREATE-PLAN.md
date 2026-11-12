# Overview

Create a phased plan in PLAN.md for a coding agent to develop a TUI-based task manager app called "taskmoth". I want to use github.com/charmbracelet/bubbletea to help build this. The app should be compatible with both MacOS and ubuntu.

Ensure the plan asks for human operator input on key design decisions when necessary. These key design decisions should always be gated.

# Design, UI, and UX

I have already designed the screens and dialogs for the application in Figma. I need you to take my designs and implement them into code, using the Figma MCP. If you do not have access to the Figma MCP, STOP before proceeding any further.

## List Views

The application should support the following main screens, called "List Views":

- Today: tasks due today. This is the default List View when starting the app. This screen can be navigated to using the key sequence `g` then `t`. To create the Today List View, implement design from Figma: @https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=32-9525&m=dev
- Inbox: tasks not assigned to a project. This screen can be navigated to using the key sequence `g` then `i`. To create the Inbox List View, implement design from Figma: @https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-18095&m=dev
- Project-specific: tasks within a specific project. A project selection dialog should appear after using the key sequence `g` then `p`.
  - For the project selection dialog, implement this design from Figma:
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-14667&m=dev
  - For the project-specific List View, implement this design from Figma: 
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-17686&m=dev
- Upcoming: tasks with a due date within the next 7 days. This screen can be navigated to using the key sequence `g` then `u`. Use a variation of the Today List View to implement this design.
- Overdue: Tasks that have an overdue due date. This screen can be navigated to using the key sequence `g` then `o`. Use a variation of the Today List View to implement this design.
- All: All tasks. This screen can be navigated to using the key sequence `g` then `a`. Use a variation of the Project-specific List View to implement this design.
- Completed: Completed tasks. This screen can be navigated to using the key sequence `g` then `c`. Use a variation of the Project-specific List View to implement this design.

## Single Task View

A single task be edited/viewed with the `e` key. For the single task view, implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=29-7123&m=dev


## Search

Search can be initiated with the `/` key. For search, Implement this design from Figma:
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-16937&m=dev


## Add a Task

Adding a task can be initiated with the `q` key. Implement this design from Figma, which should be an overlay over a view: 
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-16708&m=dev


## Other dialogs

Below are additional dialogs to implement, which are mean to overlay views:

- Edit/set task name: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-16847&m=dev
- Edit/set task description: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=15-5973&m=dev
- Edit/set task project: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=28-6404&m=dev
- Edit/set task start date: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-15310&m=dev
- Edit/set task due date: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-15226&m=dev
- Edit/set task priority: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=15-6168&m=dev
- Edit/set task duration: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-16639&m=dev
- Create new project: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-14812&m=dev
- Keyboard shortcuts: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-15079&m=dev
- Delete dialog: Implement this design from Figma.
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=40-14869&m=dev

For an example of how a dialog should be displayed over a view, see: 
@https://www.figma.com/design/uRWYa0RRojip50i2lWNdKx/Taskmoth?node-id=32-13454&m=dev

# Tasks

Tasks should have the following options:
- Name (required)
- Description (optional)
- Project (optional)
- Start date (optional)
- Due date (optional)
- Duration (optional)
- Priority (optional, options P1-P4)

Task creation should support natural language input like Todoist does. Use the rea MCP if needed to understand how Todoist handles natural language input.

# Sorting

Each List View should have the ability to sort tasks by:

- Name (default)
- Start date
- Due date
- Priority
- Duration

# Other Information

Not scoped into the initial plan but note it as a future feature: Task time tracking and task/project analytics.
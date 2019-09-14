# Project Management Software

The problem with project management software today for indie projects:

1) They’re cloud-based.
2) They’re slow, generally; written in Javascript or other web technologies.
3) They’re more complex than necessary.
4) Even if they offer a download, they might require hosting a server that works on PHP (or some other language).
5) They can’t easily be version controled.

My project will work to make a management software that resolves these issues. 

It will be native, written in Go. It will be downloadable. It will store data for each project in a text format (like JSON or something else), and will keep each project's data in a per-project location for easy VCS. It will be simple, and easy to use.

Each project is comprised of tasks. When all tasks are complete, the project is complete. The tasks will move on a grid, which can be zoomed in or out as necessary. 

Each task is a square, can be rescaled, and can contain text and images. When you zoom out enough, each task will simply become a square to represent it. Each task will use a small version of the first image embedded in the task as an icon at the top-left if it’s too small to display it.

Each project can tell you if you’re overdue for completion, and how much you’re overdue by / how many items you have to complete if you want to stay on track.

Each task has a priority, and the priority influences the importance of the task as well as which ones are served to you when the program gives you your day’s tasks to complete.

The program can give you your day’s tasks to complete, which will take into account the priority of the tasks, as well.

Each project / task has the ability to lock it to prevent editing it after creation; just completing the tasks would be available.

Each task can be completed by boolean checkmark, by filling a progress bar, by walking the task through stages, by completion of its sub-tasks, or never (in case you want a scratchpad / idea board instead of a project to manage). 

As mentioned above, tasks can be comprised of sub-tasks, in which case editing them would open up a miniature view of the parent to place tasks to complete. When all the tasks are completed, the parent is complete.

Task text might have the ability to be written in MarkDown; I'd have to find a Go MD parser, I suppose.

## To-do

~~Implement saving and loading~~
Add reading from / writing text to clipboard from textbox
Improve textbox typing
Add simple file menu (New / Clear, Save, Load, Project Settings)
Add dark theme, customizeable themes
Add sound Task
Add ability to chain sounds together by placing them next to one another
Add ability to play animated GIFs?
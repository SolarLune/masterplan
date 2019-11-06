# MasterPlan

MasterPlan is project management software for independent users. If you need to share plans across a whole company on an online site, there are tools for that. But if you just want to keep track of your todo list, version control your project plan, or make an ideaboard, MasterPlan is for you. This is just the code repository and external issue / bug tracker for MasterPlan; actual releases can be purchased from the project download page [here](https://solarlune.itch.io/masterplan?secret=fed4MHmTdQ5paAvgv4kfJzrg). (Or, you can just build it yourself from this repository if you're familiar with Go.)

## Building

TODO

## History

A few days before the initial commit to this repository, I was working on an indie game, and thought I needed a tool to help me plan out the rest of it. I asked on Twitter for some suggestions of software to try, and found that while they were solid choices, they weren't as applicable to independent development as I would have liked. Most project management software is designed for use by a large team, or even a large company.

The problems with many project management tools today are:

1) They’re cloud-based.
2) They’re slow, generally; written in Javascript or other web technologies.
3) They can be more complex than necessary to use.
4) Even if they offer a download, they might require hosting a server that works with PHP (or some other language) to serve you the management pages.
5) They can’t easily be version controled.

While these tools can be beneficial for large groups of developers, they can also become sticking points for individuals or small teams. So, I decided to make a tool myself to help independent developers plan out projects such as these. It's an ordinary, native, downloadable application that stores project data on your computer. The project plan file are plain JSON text files, and can be easily committed to a version control system. The goal for MasterPlan is to make a project management and visual planning tool that is easy to use and extremely simple. I believe it is reaching this goal.

## License

MasterPlan is licensed as All Rights Reserved, SolarLune Games 2019. 

Feel free to use the program itself and the generated plan files in the development of projects, commercial or otherwise, as that's the point of the tool, haha. You can also build MasterPlan yourself using the repo here, contribute to its development, and fork it freely, but you may not use any assets (graphics, sound, code) from this repository in any commercial creations or freely distribute builds of MasterPlan.
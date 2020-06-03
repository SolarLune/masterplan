![MasterPlan](https://user-images.githubusercontent.com/4733521/72319146-4372bd80-3653-11ea-806f-9c66c41d4c2f.png)

MasterPlan is a project management software for independent users or small teams. If you need to share plans across a whole company on an online platform, there are quite a few tools for that. However, if you just want to keep track of your todo list, version control your project plan, or make an ideaboard, MasterPlan is the tool for you.

Note that this is the code repository and external issue tracker for MasterPlan; actual releases can be purchased from the project download page [here](https://solarlune.itch.io/masterplan?secret=fed4MHmTdQ5paAvgv4kfJzrg). You can also just build it yourself from this repository if you're familiar with [Go](https://golang.org/).

## History

A few days before the initial commit to this repository, I was working on an indie game, and thought I needed a tool to help me plan out the rest of it. I asked on Twitter for some suggestions of software to try, and found that while they were solid choices, they weren't as applicable to independent development as I would have liked. Most project management software is designed for use by a large team, or even a large company. The disadvantages of many project management tools available today can be due to the features they offer, as they may not be applicable to small teams:

1) They’re cloud-based, and generally web-focused. This being the case, they usually require using a web browser to view, and so can be relatively heavy applications.
2) Because of point 1, they may not offer a downloadable copy of their software. Even if they do, they might require hosting a server that serves you the management pages, or might require owning an account to use the application.
3) They can be more complex than necessary to use by offering features that, while useful for larger organizations, are unnecessary for small teams or individual developers (for example, permissions, users, issue assignment, etc). This increases bloat and friction when it comes to learning and using the software.
4) The data you create in these tools may not easily be version controlled, as the data for your project management lies elsewhere from your source code.

While these features (cloud-based, browser apps, etc.) can be beneficial for large groups of developers, they can also become sticking points for individuals or small teams. So, I decided to make a tool myself to help independent developers plan out projects such as these. And so, the result is a simple, compiled, downloadable application that stores project data on your computer. The project plan file are plain JSON text files, and can be easily committed to a version control system. The goal for MasterPlan is to make a project management and visual planning tool that is easy to use and extremely simple. I believe it is reaching this goal.

## Building

MasterPlan is not quite fully free software, but the source is available here. If you wish to build MasterPlan or contribute to its development, I thank you, and welcome it. I've made a build script in Go to simplify the building process. The build script is located at `build_script\main.go`. The dependencies for building should be resolved automatically by `go mod` (so you should be using a recent version of Go with support for `go.mod` files). Just run:

```
> go run ./build_script/main.go -b
```

from the MasterPlan source directory to build. It should generate a folder named `bin`, and populate it with a directory with a release build for your OS and architecture. MasterPlan's `go.mod` automatically uses my fork of raylib-go, located [here](https://github.com/SolarLune/raylib-go). It should be kept up to date with `raylib-go` master, but with its `raylib/config.h` file altered to turn off automatic screenshots (as MasterPlan handles this manually).

## License

MasterPlan is copyright, All Rights Reserved, SolarLune Games 2019-2020. 

Feel free to use the program itself and the generated plan files in the development of projects, commercial or otherwise, as that's the point of the tool, haha. You can also build MasterPlan yourself using the repo here, contribute to its development, and fork it freely, but you may not use any assets (graphics files, sound files, code, etc) from this repository in any commercial creations. **Please also do not distribute builds of MasterPlan to others.**

Special thanks to raysan5 for creating Raylib, as it's pretty key to the development of MasterPlan, and works rather well!

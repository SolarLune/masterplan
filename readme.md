# MasterPlan

MasterPlan is project management software for independent users. If you need to share plans across a whole company on an online site, there are tools for that. But if you just want to keep track of your todo list, version control your project plan, or make an ideaboard, MasterPlan is for you. This is just the code repository and external issue / bug tracker for MasterPlan; actual releases can be purchased from the project download page [here](https://solarlune.itch.io/masterplan?secret=fed4MHmTdQ5paAvgv4kfJzrg). (Or, you can just build it yourself from this repository if you're familiar with Go.)

## History

A few days before the initial commit to this repository, I was working on an indie game, and thought I needed a tool to help me plan out the rest of it. I asked on Twitter for some suggestions of software to try, and found that while they were solid choices, they weren't as applicable to independent development as I would have liked. Most project management software is designed for use by a large team, or even a large company.

The problems with many project management tools today are:

1) They’re cloud-based.
2) They’re slow, generally; written in Javascript or other web technologies.
3) They can be more complex than necessary to use.
4) Even if they offer a download, they might require hosting a server that works with PHP (or some other language) to serve you the management pages.
5) They can’t easily be version controled.

While these tools can be beneficial for large groups of developers, they can also become sticking points for individuals or small teams. So, I decided to make a tool myself to help independent developers plan out projects such as these. It's an ordinary, native, downloadable application that stores project data on your computer. The project plan file are plain JSON text files, and can be easily committed to a version control system. The goal for MasterPlan is to make a project management and visual planning tool that is easy to use and extremely simple. I believe it is reaching this goal.

## Building

MasterPlan is not quite fully free software, but the source is open. If you wish to build MasterPlan or contribute to its development, I thank you and welcome it.

I've made a build script in Golang to make building MasterPlan easier. The build script is located at `build_script\main.go`. Run it with an argument of `-b` to build. The dependencies for building should be resolved automatically by `go mod` (so you should be using a Golang build that has support for `go.mod` files).

However, note that in order to build, you will also need to have my custom Raylib-go repo downloaded and sitting next to MasterPlan's directory. Specifically, it's [this](https://github.com/SolarLune/raylib-go/tree/ImgFormats) branch of the repo, known as `ImgFormats`. 

This particular branch is up-to-date with `raylib-go` master, but has had its `raylib/config.h` file altered to build with support for additional image file formats (like JPEG). With the `ImgFormats` branch cloned and sitting in the folder `raylib-go-solarlune`, next to the MasterPlan source directory, it's easy to build. Just run:

```
> go run ./build_script/main.go -b
```

from the MasterPlan source directory. It should generate a folder named `bin`, and populate it with a directory with a release build for your OS and architecture.

## License

MasterPlan is licensed as All Rights Reserved, SolarLune Games 2019. 

Feel free to use the program itself and the generated plan files in the development of projects, commercial or otherwise, as that's the point of the tool, haha. You can also build MasterPlan yourself using the repo here, contribute to its development, and fork it freely, but you may not use any assets (graphics files, code, etc) from this repository in any commercial creations. Please also do not distribute builds of MasterPlan to others.

Special thanks to raysan5 for creating Raylib, as it's pretty key to the development of MasterPlan.
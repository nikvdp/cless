# `cless` - `less` with colors

Simple utility to do one thing: let you scroll colored output in less **even if
the tool you're using tries to disable it** when it detects it's being piped
into another command
 

### Example:

Normally when you run `ls` with `--color=auto` it will disable colorization if
it detects it's output is being piped somewhere (ie `less`):

```shell
ls --color=auto | less  # no colors
```

If you run it in `cless`, cless fools it into thinking it's output is not going
anywhere and then pipes into `less` for you:

```shell
cless ls --color=auto  # output will be colored
```

(unfortunately usage can't be exactly the same as with normal `less` because
`cless` can only do it's magic if it can control how the process is started)


### misc

there's also a python version of `cless` in the `python` branch



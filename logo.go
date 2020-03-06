package main

import "fmt"

const logo = `

 ____ o__ __o____   o         o        o__ __o        o__ __o
   /   \   /   \   <|>       <|>      /v     v\      <|     v\ 
        \o/        < >       < >     />       <\     / \     <\
         |          |         |    o/           \o   \o/     o/
        < >         o__/_ _\__o   <|             |>   |__  _<|
         |          |         |    \\           //    |       \
         o         <o>       <o>     \         /     <o>       \o
        <|          |         |       o       o       |         v\
        / \        / \       / \      <\__ __/>      / \         <\

                             %10v

`

func printProlog() {
    fmt.Printf(logo, ApplicationVersion)
}

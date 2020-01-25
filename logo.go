package main

import "fmt"

const logo = `
  ,:\      /:.                                                                                ,:\      /:.
 //  \_()_/  \\                                                                              //  \_()_/  \\
||   |    |   ||                                                                            ||   |    |   ||   
||   |    |   ||     ____ o__ __o____   o         o        o__ __o        o__ __o           ||   |    |   ||
||   |____|   ||       /   \   /   \   <|>       <|>      /v     v\      <|     v\          ||   |____|   ||
 \\  / || \  //             \o/        < >       < >     />       <\     / \     <\          \\  / || \  //
  ':/  ||  \;'               |          |         |    o/           \o   \o/     o/           ':/  ||  \;'
       ||                   < >         o__/_ _\__o   <|             |>   |__  _<|                 ||
       ||                    |          |         |    \\           //    |       \                ||
       XX                    o         <o>       <o>     \         /     <o>       \o              XX
       XX                   <|          |         |       o       o       |         v\             XX
       XX                   / \        / \       / \      <\__ __/>      / \         <\            XX
       XX                                                                                          XX
       OO                                           %10v                                     OO

`

func printProlog() {
    fmt.Printf(logo, ApplicationVersion)
}

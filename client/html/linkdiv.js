// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
import {iconClassName} from "./utils.js"
import { activateSelected, selectDiv} from "./navigation.js"
import { div, img, a, frag } from "./elements.js"

const aOnClick = e => e.preventDefault()
const click = e => selectDiv(e.currentTarget)

let LinkDiv = ({link, dblClick}) => {
    return ( 
        div(
            {className: "link", onClick: click, onDoubleClick: dblClick},
            div({}, link.icon && img({src:link.icon, className: iconClassName(link.profile), height:"20", width:"20"})),
            a(
                {href: link.href, rel: link.rel, tabIndex: -1, onClick: aOnClick }, 
                link.title
            )
        )
    )
}

export let linkDiv = (link, dblClick) => React.createElement(LinkDiv, {link: link, dblClick: dblClick}) 



// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
import {doDelete, doPost, iconClassName} from "../common/utils.js"
import { img, a, span } from "../common/elements.js"

export let link = (link, comment, dismiss, move) => {

    let activate = (href, close, profile) => {
        doPost(href).then(response => {
                response.ok && close && dismiss("window" !== profile && "browsertab" !== profile , "browsertab" !== profile)
            })
    }

    let  onKeyDown = event => {
        let { key, ctrlKey, shiftKey, altKey} = event;
        if (key === "ArrowRight") {
            if (event.target.rel === "related") {
                move("right", event.target.href);
            }
        }  else if (key === "ArrowUp" || key === "k" && ctrlKey || key === 'Tab' && shiftKey && !ctrlKey && !altKey) {
            move("up");
        } else if (key === "ArrowDown" || key === "j" && ctrlKey || key === 'Tab' && !shiftKey && !ctrlKey && !altKey) {
            move("down");
        } else if (key === "Enter") {
            activate(event.target.href, !ctrlKey, event.target.dataset.profile) 
        } else if (key === "Delete") {
            doDelete(event.target.href).then(response => response.ok && !ctrlKey && dismiss(true, true))
        } else { 
            return;
        }
        event.preventDefault();
    }

    comment = comment || "" 
    return a({  className: "link", 
                onClick: e => e.preventDefault(),
                onDoubleClick: e => {
                    activate(e.currentTarget.href, true, e.currentTarget.dataset.profile)
                    e.preventDefault()
                },
                onKeyDown: onKeyDown,
                rel:link.rel, 
                href: link.href,
                tabIndex: -1,
                "data-profile": link.profile,
             }, 
        link.icon && img({className: "icon", src:link.icon, height:"20", width:"20"}), 
        span({className:"title"}, link.title),
        span({className:"comment"}, comment)
    )
}


export const iconClassName = profile => "icon" + ("window" === profile ? " window" : "")

export const getJson = (href, handler) =>
    fetch(href)
        .then(resp => resp.json())
        .then(json => handler(json), error => console.warn(error))


export const doPost = (href, params) => {
    if (params) {
        let separator = href.indexOf('?') > -1 ? '&' : '?'
        for (const [name, val] of Object.entries(params)) {
            href = href + separator + name + "=" + val
            separator = "&"
        }
    }
    return fetch(href, { method: "POST" })
}

export const doDelete = href => fetch(href, { method: "DELETE" })

export let linkHref = (res, rel) => {
    rel = rel || "self"
    return res._links.find(l => l.rel === rel)?.href
}
export let menuHref = res => linkHref(res, "org.refude.menu")


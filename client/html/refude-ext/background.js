let tabsSocket
let commandSocket

const reportTabs = () => {
    console.log("reportTabs")
    chrome.tabs.query({}, tabs => {
        let tabsData = tabs.map(t => {
            return {
                id: "" + t.id,
                title: t.title,
                url: t.url,
                favIcon: t.favIconUrl
            }
        })
        fetch("http://localhost:7938/tab/", {method: "POST", body: JSON.stringify(tabsData)})
            .then(response => {
                if (!response.ok) {
                    throw new Error(response.status)
                }
            })
            .catch(e => { // If we couldn't deliver data, try again i 5 secs.
                setTimeout(reportTabs, 5000)
            })
    })
}

const watch = () => {
    // commands
    const focusTab = "focustab"
    reportTabs()
    let evtSource = new EventSource("http://localhost:7938/watch")
    evtSource.onopen = reportTabs
    evtSource.addEventListener("showLauncher", showLauncher)
    evtSource.addEventListener("hideLauncher", hideLauncher)
    evtSource.addEventListener("restoreTab", restoreTab)
    evtSource.addEventListener("focusTab", ({data}) => {
        let tabId = parseInt(data)
        tabId && chrome.tabs.update(tabId, { 'active': true }, (tab) => { }) 
    })
    evtSource.onerror = error => {
        console.log(error)
        if (evtSource.readyState === 2) {
            setTimeout(watch, 5000)
        }
    } 

}


let rememberedTab

let showLauncher = () => {
    chrome.windows.getCurrent({}, window => {
        if (!window) {
            chrome.windows.create({ focused: true, url: "http://localhost:7938/refude/html/launcher/" })
        } else {
            chrome.tabs.query({ active: true }, ([tab]) => {
                rememberedTab = tab
                chrome.tabs.query(
                    { url: "http://localhost:7938/refude/html/launcher/" },
                    tabs => chrome.tabs.remove(tabs.map(t => t.id), () => {
                        chrome.tabs.create({ active: true, index: 0, url: "http://localhost:7938/refude/html/launcher" })
                    }))
            })
        }
    })
}

let restoreTab = () => {
    rememberedTab && chrome.tabs.update(rememberedTab.id, { active: true })
}

let hideLauncher = () => {
    chrome.tabs.query(
        { url: "http://localhost:7938/refude/html/launcher/" }, tabs => chrome.tabs.remove(tabs.map(t => t.id))
    )
}


/*
    Some nonsense one has to do to keep the service worker alive when on manifest version 3
    Stupid. And there seems to be no way of keeping alive if server is down (reconnect attempts does not 
        extend lifetime)
    Sticking to manifest v2 as long as possible

const keepAlive = () => {
    let ps = new WebSocket("ws://localhost:7938/ping")
    let ping = () => {
        if (ps) {
            console.log('pinging...')
            ps.send("ping")
            setTimeout(ping, 10000)
        }
    }
    ps.onopen = () => {
        console.log("start pinging")
        ping()
    }
    ps.onclose = () => {
        ps.close()
        ps = null
        setTimeout(keepAlive, 5000)
    }
}
keepAlive()
*/

console.log("Starting")
chrome.tabs.onRemoved.addListener(reportTabs)
chrome.tabs.onUpdated.addListener(reportTabs)
watch()

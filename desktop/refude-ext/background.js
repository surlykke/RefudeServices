let tabsSocket
let commandSocket

const reportTabs = () => {
    chrome.tabs.query({}, tabs => {
		let tabsData = tabs.map(t => {
			return {
				id: "" + t.id,
				title: t.title,
				url: t.url,
				favIcon: t.favIconUrl
			}
		})
		fetch("http://localhost:7938/tabsink", { method: "POST", body: JSON.stringify(tabsData) })
			.then(response => {
				if (!response.ok) {
					throw new Error(response.status)
				}
			})
			.catch(() => { // If we couldn't deliver data, try again i 5 secs.
				setTimeout(reportTabs, 5000)
			})
	})
}

const watch = () => {
	let evtSource = new EventSource("http://localhost:7938/watch")
	evtSource.onopen = reportTabs
	evtSource.addEventListener("showDesktop", showDesktop)
	evtSource.addEventListener("restoreTab", restoreTab)
	evtSource.addEventListener("focusTab", ({ data }) => {
		let tabId = parseInt(data)
		tabId && chrome.tabs.update(tabId, { 'active': true }, () => { })
	})
	evtSource.addEventListener("closeTab", ({ data }) => {
		console.log("closeTab", data)
		let tabId = parseInt(data)
		tabId && chrome.tabs.remove(tabId)
	})
	evtSource.onerror = error => {
		console.log(error)
		if (evtSource.readyState === 2) {
			setTimeout(watch, 5000)
		}
	}
}

let rememberedTab

let showDesktop = () => {
	console.log("showDesktop")
	chrome.windows.getCurrent({}, window => {
		if (!window) {
			chrome.windows.create({ focused: true, url: "http://localhost:7938/desktop/" })
		} else {
			chrome.tabs.query({ active: true }, ([tab]) => {
				rememberedTab = tab
				chrome.tabs.query(
					{ url: "http://localhost:7938/desktop/*" },
					tabs => {
						if (tabs.length == 0) {
							chrome.tabs.create({ active: true, index: 0, url: "http://localhost:7938/desktop/" })
						} else {
							chrome.tabs.update(tabs[0].id, { active: true })
							chrome.windows.update(tabs[0].windowId, { focused: true })
							chrome.tabs.remove(tabs.slice(1).map(t => t.id))
						}
					}
				)
			})
		}
	})
}


let restoreTab = () => {
	rememberedTab && chrome.tabs.update(rememberedTab.id, { active: true })
}


/*
	Some nonsense one has to do to keep the service worker alive when on manifest version 3.
    
	Stupid. 
    
	And there seems to be no way of keeping alive if server is down (reconnect attempts does not extend lifetime)

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

reportTabs()
chrome.tabs.onRemoved.addListener(reportTabs)
chrome.tabs.onUpdated.addListener(reportTabs)
watch()

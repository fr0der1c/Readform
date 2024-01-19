package main

import (
	"time"
)

var domainAgentDict = make(map[string]*WebsiteAgent)

// mainLoop creates newly enabled agents and close newly disabled agents.
func mainLoop() {
	agents := make(map[string]*WebsiteAgent) // enabled agents map

	for {
		enabledAgents := currentConf.EnabledWebsites
		if len(enabledAgents) == 0 {
			logger.Warning("[Main] No enabled website.")
		}

		// add newly enabled agents
		for _, agentName := range enabledAgents {
			if _, exists := agents[agentName]; exists {
				continue
			}

			found := false
			var a *WebsiteAgent
			for _, agent := range allAgents {
				if agent.Name() == agentName {
					found = true
					a = agent
				}
			}
			if !found {
				logger.Errorf("[Main] Unknown website agent name %s, ignoring it", agentName)
				continue
			}

			logger.Infof("[Main] Enabling agent %s...", agentName)
			ctx, cancelFunc := GetBrowserCtx()

			agentConf := currentConf.AgentConfs[a.Name()]
			err := a.Init(ctx, cancelFunc, agentConf)
			if err != nil {
				logger.Errorf("[Main] init agent %s failed: %v", a.Name(), err)
				continue
			}

			agents[agentName] = a
			for _, domain := range a.BaseDomains() {
				domainAgentDict[domain] = a
			}
			a.StartRefreshingRSS() // a goroutine will be spawned to periodically refresh RSS and save HTML content to local.
			time.Sleep(1 * time.Second)
		}

		// close newly disabled agents
		var toDel []string
		for agentName := range agents {
			if ContainsStringSlice(enabledAgents, agentName) {
				continue
			}
			toDel = append(toDel, agentName)
		}
		for _, agentName := range toDel {
			logger.Infof("[Main] Disabling agent %s...", agentName)
			a := agents[agentName]
			err := a.Shutdown()
			if err != nil {
				logger.Errorf("[Main] shutdown agent %s failed: %v", agentName, err)
			}
			delete(agents, agentName)
		}

		time.Sleep(1 * time.Second)
	}
}

func main() {
	initLogger()
	initConf()
	initDB()
	startSaver()
	go mainLoop()
	runHTTPServer()
}

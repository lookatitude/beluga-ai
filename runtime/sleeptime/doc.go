// Package sleeptime provides sleep-time compute capabilities for the Beluga AI
// framework. It allows agents to perform background work during idle periods,
// such as memory reorganization and contradiction resolution.
//
// Sleep-time compute detects when an agent session is idle (no active user
// interaction) and schedules background tasks that improve the agent's
// performance and knowledge consistency. Tasks are preempted when the agent
// wakes up (user returns).
//
// # Components
//
// [IdleDetector] determines whether a session is idle. The default
// [InactivityDetector] uses a configurable inactivity timeout.
//
// [Task] defines a unit of background work. Built-in tasks include memory
// reorganization and contradiction resolution. Custom tasks can be registered
// via the task registry.
//
// [Scheduler] orchestrates task execution during idle periods. It polls the
// idle detector, runs eligible tasks with bounded concurrency, and preempts
// tasks when the session wakes up.
//
// [SleeptimePlugin] integrates the scheduler with the runtime plugin system,
// waking on BeforeTurn and updating session state on AfterTurn.
//
// # Usage
//
//	scheduler := sleeptime.NewScheduler(detector,
//	    sleeptime.WithTasks(memReorgTask, contradictionTask),
//	    sleeptime.WithMaxConcurrentTasks(2),
//	    sleeptime.WithPollInterval(5 * time.Second),
//	)
//
//	plugin := sleeptime.NewSleeptimePlugin(scheduler)
//	runner := runtime.NewRunner(myAgent, runtime.WithPlugins(plugin))
package sleeptime

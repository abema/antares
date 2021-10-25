package core

type (
	OnDownloadHandler  func(file *File)
	OnReportHandler    func(reports Reports)
	OnTerminateHandler func()
)

func MergeOnDownloadHandlers(handlers ...OnDownloadHandler) OnDownloadHandler {
	return func(file *File) {
		for _, handler := range handlers {
			handler(file)
		}
	}
}

func MergeOnReportHandlers(handlers ...OnReportHandler) OnReportHandler {
	return func(reports Reports) {
		for _, handler := range handlers {
			handler(reports)
		}
	}
}

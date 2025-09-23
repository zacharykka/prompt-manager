export function LoadingScreen() {
  return (
    <div className="flex h-screen w-full items-center justify-center bg-slate-50 text-slate-500">
      <div className="flex flex-col items-center gap-3">
        <div className="h-10 w-10 animate-spin rounded-full border-4 border-brand-200 border-t-brand-500" />
        <p className="text-sm font-medium">加载中，请稍候...</p>
      </div>
    </div>
  )
}

import Stats from './ServerStats'
import type { ServerStats } from '../../schemas'

export default function QueryResults({ error, isPending, isLoading, data }: { error?: Error | null, isPending: boolean, isLoading: boolean, data?: ServerStats }) {
  if (error) {
    return <p className="text-red-200">Failed to get statistics</p>
  }
  if (isPending && !isLoading) {
    return
  }
  return (
    <>
      <div className="grid grid-cols-2 gap-3 p-2 bg-[#20252e] border border-[#444850] rounded-md">
        <Stats stats={data?.Improved} type="Improved" game={data?.Game ?? ''} />
        <Stats stats={data?.Rising} type="Rising" game={data?.Game ?? ''} />
      </div>
    </>
  )
}
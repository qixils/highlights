import type { ServerStat, ServerStats } from '../../schemas'
import Stat from './ServerStat'

export default function ServerStats({ stats, type, game }: { stats: ServerStat[] | undefined, type: Exclude<keyof ServerStats, 'Game'>, game: string }) {
  const idxs = Array.from({ length: 5 }, (_, i) => i);

  const target = new Date()
  target.setDate(0)
  const targetMonth = target.toLocaleString('en-US', { month: 'long' })

  // const prev = new Date(target)
  // prev.setDate(0)
  // const prevMonth = prev.toLocaleString('en-US', { month: 'long' })

  return (
    <>
      <div className='grid grid-rows-7 gap-2 overflow-hidden'>
        <p className="text-center font-bold">
          {type === 'Improved' ? 'Most Improved' : 'Rising Stars'}
        </p>
        <p className="text-center font-semibold text-sm">
          {type === 'Improved' ? `Greatest time saves of ${targetMonth}` : `Fastest new runners of ${targetMonth}`}
        </p>

        {idxs.map(idx => <Stat stat={stats?.[idx]} animate={stats === undefined} type={type} game={game} key={idx} />)}
      </div>
    </>
  )
}
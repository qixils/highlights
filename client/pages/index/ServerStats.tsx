import type { ServerStat, ServerStats } from '../../schemas'
import Stat from './ServerStat'

export default function ServerStats({ stats, type, game }: { stats: ServerStat[] | undefined, type: Exclude<keyof ServerStats, 'Game'>, game: string }) {
  const idxs = Array.from({ length: 5 }, (_, i) => i);
  return (
    <>
      <div className='grid grid-rows-6 gap-2 overflow-hidden'>
        <p className="text-center font-bold">{type === 'Improved' ? 'Most Improved' : 'Rising Stars'}</p>
        {idxs.map(idx => <Stat stat={stats?.[idx]} animate={stats === undefined} type={type} game={game} key={idx} />)}
      </div>
    </>
  )
}
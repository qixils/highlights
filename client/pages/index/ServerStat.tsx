import type { ServerStat, ServerStats } from '../../schemas'

export default function ServerStat({ stat, animate, type, game }: { stat?: ServerStat, animate?: boolean, type: Exclude<keyof ServerStats, 'Game'>, game: string }) {
  let timeStat: string | undefined = undefined
  if (stat) {
    const sec = stat.Statistic % 60
    const min = Math.floor(stat.Statistic / 60.0)

    timeStat = `${min}:${sec.toLocaleString('en-US', { minimumIntegerDigits: 2, useGrouping: false })}`

    if (type === 'Improved') {
      timeStat = `-${timeStat}`
    }
  }

  const url = !stat ? '' : `https://speedrun.com/${game}/runs/${stat.Url}`

  return (
    <>
      <div className="flex-1 flex flex-row items-center gap-2 overflow-hidden">
        {
          stat?.Picture
            ? <img src={stat.Picture} className="rounded-full size-5" />
            : <div className={`rounded-full bg-slate-500 size-5 ${animate ? 'animate-pulse' : ''}`}></div>
        }
        {
          stat || !animate
            ? <p className={`flex-1 font-medium truncate ${!stat ? 'text-[#aaa]' : ''}`}>{ stat?.Name || 'N/A' }</p>
            : <>
              <div className={`flex-1 rounded-md bg-slate-500 h-2 animate-pulse`}></div>
            </>
        }
        <div className='flex flex-row items-center justify-end'>
          {
            timeStat !== undefined
              ? <a href={url} className="font-medium text-green-600 underline">{ timeStat }</a>
              : <div className={`rounded-md bg-slate-500 h-2 w-12 ${animate ? 'animate-pulse' : ''}`}></div>
          }
        </div>
      </div>
    </>
  )
}
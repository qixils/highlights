import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from "react";
import QueryResults from "./QueryResults";
import { usePageContext } from "vike-react/usePageContext";
import { ServerStatsSchema } from "../../schemas";

const urlMatcher = /speedrun\.com\/([a-zA-Z0-9_-]+).*\?h=([a-zA-Z0-9_]+)/

export default function Page() {
  const { urlOriginal } = usePageContext()

  const [url, setUrl] = useState('')
  const [pendingUrl, setPendingUrl] = useState('')

  const [, game, category] = urlMatcher.exec(url) ?? []
  const [, pendingGame, pendingCategory] = urlMatcher.exec(pendingUrl) ?? []

  const { data, isPending, isLoading, error } = useQuery({
    queryKey: [game, category],
    queryFn: async () => {
      if (!game || !category) return null

      // http://localhost:30100
      const url = new URL("https://highlights.speedrun.club/api/v1/highlights")
      url.searchParams.set("game", game)
      url.searchParams.set("leaderboard", category)
      const stats = ServerStatsSchema.parse(await fetch(url).then(r => r.json()))
      return stats
    },
  })

  async function tryRefetch() {
    if (!pendingGame) return
    if (!pendingCategory) return

    setUrl(pendingUrl)

    const url = new URL(window.location.href)
    url.searchParams.set("game", pendingGame)
    url.searchParams.set("category", pendingCategory)
    window.history.pushState({}, "", url.href)
  }

  async function tryEnter(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key !== 'Enter') return
    await tryRefetch()
  }

  useEffect(() => {
    const params = new URL(window?.location?.href || urlOriginal).searchParams
    
    const game = params.get("game")
    const category = params.get("category")
    if (!game || !category) return

    const newUrl = `https://speedrun.com/${game}?h=${category}`
    setPendingUrl(newUrl)
    setUrl(newUrl)
  }, [])

  return (
    <>
      <div className="flex flex-col justify-center gap-2">
        <h1 className="text-xl font-bold text-center">Speedrun Highlights Club</h1>
        <p className="text-lg font-medium text-center">View fun community statistics for leaderboards on speedrun.com</p>
      </div>

      <div className="grid grid-cols-[1fr_auto] gap-2 items-stretch">
        <input name="url" value={pendingUrl} onChange={e => setPendingUrl(e.target.value)} onKeyUp={e => tryEnter(e)} placeholder="Speedrun.com Leaderboard URL" className="border border-[#444850] rounded-md px-2 py-1.5"></input>
        <button disabled={!pendingGame || !pendingCategory} onClick={e => tryRefetch()} className="border border-[#dd22aa] enabled:hover:bg-[#111122] enabled:hover:brightness-125 px-2 py-1.5 rounded-md flex flex-row items-center disabled:opacity-50">
          <span>View Stats</span>
        </button>
      </div>
      
      <QueryResults error={error} isPending={isPending} isLoading={isLoading} data={data} />
    </>
  );
}

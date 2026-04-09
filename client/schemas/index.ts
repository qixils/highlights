import z from "zod"

export const ServerStatSchema = z.object({
  Name: z.string(),
  Picture: z.string(),
  Url: z.string(),
  Statistic: z.number(),
})

export type ServerStat = z.output<typeof ServerStatSchema>

export const ServerStatsSchema = z.object({
  Game: z.string(),
  Improved: z.array(ServerStatSchema),
  Rising: z.array(ServerStatSchema),
})

export type ServerStats = z.output<typeof ServerStatsSchema>
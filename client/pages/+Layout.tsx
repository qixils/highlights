export { Layout }
 
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import './Layout.css'

const client = new QueryClient()
 
function Layout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <QueryClientProvider client={client}>
        <div className="h-dvh w-dvw flex flex-col items-center justify-between overflow-y-scroll overflow-x-hidden text-white bg-[#000011]">
          <div className="flex flex-col justify-center items-stretch p-3 gap-5 max-w-2xl w-full overflow-hidden">
            {children}
          </div>

          <div className="flex flex-col justify-end items-stretch p-3 gap-2 max-w-2l text-center text-[#777]">
            <p>Made with ❤️ by Lexi</p>
            <p>Source code available on <a href="https://github.com/qixils/highlights" className="text-[#aaa] underline">GitHub</a></p>
          </div>
        </div>
      </QueryClientProvider>
    </>
  )
}
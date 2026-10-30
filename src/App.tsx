import { BrowserRouter, Navigate, Route, Routes } from 'react-router'
import { MotionConfig } from 'motion/react'
import { StoreProvider } from './lib/store'
import { Layout } from './components/Layout'
import { LotsPage } from './pages/Lots'
import { LotPage } from './pages/LotPage'
import { GiftPage, GiftsPage } from './pages/Gifts'
import { InventoryPage, ProfilePage } from './pages/Account'
import { ConnectPage } from './pages/Connect'

export function App() {
  return (
    <MotionConfig reducedMotion="user">
      <StoreProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<Navigate to="/usernames" replace />} />
              <Route path="usernames" element={<LotsPage kind="username" />} />
              <Route path="usernames/:slug" element={<LotPage kind="username" />} />
              <Route path="numbers" element={<LotsPage kind="number" />} />
              <Route path="numbers/:slug" element={<LotPage kind="number" />} />
              <Route path="gifts" element={<GiftsPage />} />
              <Route path="gifts/:id" element={<GiftPage />} />
              <Route path="inventory" element={<InventoryPage />} />
              <Route path="profile" element={<ProfilePage />} />
              <Route path="connect" element={<ConnectPage />} />
              <Route path="*" element={<Navigate to="/usernames" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </StoreProvider>
    </MotionConfig>
  )
}

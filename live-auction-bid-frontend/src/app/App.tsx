import { HostConsolePage } from '../pages/host-console/HostConsolePage';
import { LiveRoomPage } from '../pages/live-room/LiveRoomPage';

export function App() {
  return location.pathname.startsWith('/host') ? <HostConsolePage /> : <LiveRoomPage />;
}

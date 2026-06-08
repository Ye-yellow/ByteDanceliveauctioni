import { Suspense, lazy, useEffect, useState } from 'react';
import type { ComponentType } from 'react';
import { SPA_NAVIGATE_EVENT } from '../shared/navigation';

const HomePage = lazy(() => import('../pages/HomePage').then((module) => ({ default: module.HomePage })));
const HomeLivePage = lazy(() => import('../pages/HomeLivePage').then((module) => ({ default: module.HomeLivePage })));
const LiveRoomPage = lazy(() => import('../pages/LiveRoomPage').then((module) => ({ default: module.LiveRoomPage })));
const ResultPage = lazy(() => import('../pages/ResultPage').then((module) => ({ default: module.ResultPage })));
const HistoryPage = lazy(() => import('../pages/HistoryPage').then((module) => ({ default: module.HistoryPage })));
const AuthorProfilePage = lazy(() => import('../pages/AuthorProfilePage').then((module) => ({ default: module.AuthorProfilePage })));
const ShopPage = lazy(() => import('../pages/ShopPage').then((module) => ({ default: module.ShopPage })));
const MessagePage = lazy(() => import('../pages/MessagePage').then((module) => ({ default: module.MessagePage })));
const PublishPage = lazy(() => import('../pages/PublishPage').then((module) => ({ default: module.PublishPage })));
const LoginPage = lazy(() => import('../pages/LoginPage').then((module) => ({ default: module.LoginPage })));
const SearchPage = lazy(() => import('../pages/SearchPage'));
const MusicPage = lazy(() => import('../pages/MusicPage'));
const MusicRankPage = lazy(() => import('../pages/MusicRankPage'));
const ShopDetailPage = lazy(() => import('../pages/ShopDetailPage').then((module) => ({ default: module.ShopDetailPage })));
const VideoDetailPage = lazy(() => import('../pages/VideoDetailPage').then((module) => ({ default: module.VideoDetailPage })));
const ReportPage = lazy(() => import('../pages/ReportPage').then((module) => ({ default: module.ReportPage })));
const SubmitReportPage = lazy(() => import('../pages/SubmitReportPage').then((module) => ({ default: module.SubmitReportPage })));
const MessageAllPage = lazy(() => import('../pages/MessageAllPage').then((module) => ({ default: module.MessageAllPage })));
const MessageChatPage = lazy(() => import('../pages/MessageChatPage').then((module) => ({ default: module.MessageChatPage })));
const MessagePeoplePage = lazy(() => import('../pages/MessagePeoplePage').then((module) => ({ default: module.MessagePeoplePage })));
const MessageFansPage = lazy(() => import('../pages/MessagePeoplePage').then((module) => ({ default: module.MessageFansPage })));
const MessageVisitorsPage = lazy(() => import('../pages/MessagePeoplePage').then((module) => ({ default: module.MessageVisitorsPage })));
const MessageSharePage = lazy(() => import('../pages/MessageSharePage').then((module) => ({ default: module.MessageSharePage })));
const OtherLoginPage = lazy(() => import('../pages/LoginFlowPages').then((module) => ({ default: module.OtherLoginPage })));
const PasswordLoginPage = lazy(() => import('../pages/LoginFlowPages').then((module) => ({ default: module.PasswordLoginPage })));
const VerificationCodePage = lazy(() => import('../pages/LoginFlowPages').then((module) => ({ default: module.VerificationCodePage })));
const RetrievePasswordPage = lazy(() => import('../pages/LoginFlowPages').then((module) => ({ default: module.RetrievePasswordPage })));
const LoginHelpPage = lazy(() => import('../pages/LoginFlowPages').then((module) => ({ default: module.LoginHelpPage })));
const CountryChoosePage = lazy(() => import('../pages/LoginFlowPages').then((module) => ({ default: module.CountryChoosePage })));
const EditUserInfoPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.EditUserInfoPage })));
const EditUserInfoItemPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.EditUserInfoItemPage })));
const AddSchoolPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.AddSchoolPage })));
const ChooseSchoolPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.ChooseSchoolPage })));
const ChooseDepartmentPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.ChooseDepartmentPage })));
const DisplayTypePage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.DisplayTypePage })));
const ChooseLocationPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.ChooseLocationPage })));
const ChooseProvincePage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.ChooseProvincePage })));
const ChooseCityPage = lazy(() => import('../pages/ProfileEditPages').then((module) => ({ default: module.ChooseCityPage })));
const MyCardPage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.MyCardPage })));
const RequestUpdatePage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.RequestUpdatePage })));
const MyRequestUpdatePage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.MyRequestUpdatePage })));
const ProfileSettingReplicaPage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.ProfileSettingReplicaPage })));
const LookHistoryPage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.LookHistoryPage })));
const DeclareSchoolPage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.DeclareSchoolPage })));
const MinorProtectionPage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.MinorProtectionPage })));
const MinorProtectionDetailPage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.MinorProtectionDetailPage })));
const TriggerTimePage = lazy(() => import('../pages/ProfileUtilityPages').then((module) => ({ default: module.TriggerTimePage })));
type NoticeGroup = 'helper' | 'system' | 'task' | 'live' | 'money';

const lazyUtility = <P extends object>(name: string) => lazy(() =>
  import('../pages/DouyinUtilityPages').then((module) => ({ default: module[name as keyof typeof module] as ComponentType<P> })),
);

const AlbumDetailPage = lazyUtility<Record<string, never>>('AlbumDetailPage');
const ServiceProtocolPage = lazyUtility<Record<string, never>>('ServiceProtocolPage');
const NoticePage = lazyUtility<{ title: string; group: NoticeGroup }>('NoticePage');
const NoticeSettingPage = lazyUtility<Record<string, never>>('NoticeSettingPage');
const ChatDetailPage = lazyUtility<Record<string, never>>('ChatDetailPage');
const RedPacketPage = lazyUtility<Record<string, never>>('RedPacketPage');
const MoreSearchPage = lazyUtility<Record<string, never>>('MoreSearchPage');
const JoinedGroupChatPage = lazyUtility<Record<string, never>>('JoinedGroupChatPage');
const PeopleListPage = lazyUtility<{ title?: string; desc?: string }>('PeopleListPage');
const ScanPage = lazyUtility<Record<string, never>>('ScanPage');
const RemarkPage = lazyUtility<Record<string, never>>('RemarkPage');

function matchesRoute(path: string, route: string): boolean {
  return path === route || path.startsWith(`${route}/`);
}

function routeForPath(path: string) {
  const roomMatch = path.match(/^\/m\/room\/([^/]+)/);
  if (roomMatch) return <LiveRoomPage roomId={decodeURIComponent(roomMatch[1])} />;
  if (path.startsWith('/m/result/')) return <ResultPage />;
  if (matchesRoute(path, '/m/history')) return <HistoryPage />;

  if (path === '/home/search') return <SearchPage />;
  if (path === '/home/live') return <HomeLivePage />;
  if (path === '/home/music') return <MusicPage />;
  if (path === '/home/music-rank-list') return <MusicRankPage />;
  if (path === '/home/report') return <ReportPage />;
  if (path === '/home/submit-report') return <SubmitReportPage />;
  if (path === '/video-detail') return <VideoDetailPage />;
  if (path === '/user') return <AuthorProfilePage />;
  if (path === '/album-detail') return <AlbumDetailPage />;
  if (path === '/service-protocol') return <ServiceProtocolPage />;

  if (path === '/shop/detail') return <ShopDetailPage />;
  if (matchesRoute(path, '/shop') || matchesRoute(path, '/m/shop')) return <ShopPage />;

  if (path === '/message/chat/red-packet-detail') return <RedPacketPage />;
  if (path === '/message/chat/detail') return <ChatDetailPage />;
  if (path === '/message/chat') return <MessageChatPage />;
  if (path === '/message/share-to-friend') return <MessageSharePage />;
  if (path === '/message/all') return <MessageAllPage />;
  if (path === '/message/more-search') return <MoreSearchPage />;
  if (path === '/message/joined-group-chat') return <JoinedGroupChatPage />;
  if (path === '/message/fans') return <MessageFansPage />;
  if (path === '/message/visitors') return <MessageVisitorsPage />;
  if (path === '/message/douyin-helper') return <NoticePage title="抖音小助手" group="helper" />;
  if (path === '/message/system-notice') return <NoticePage title="系统通知" group="system" />;
  if (path === '/message/task-notice') return <NoticePage title="任务通知" group="task" />;
  if (path === '/message/live-notice') return <NoticePage title="直播通知" group="live" />;
  if (path === '/message/money-notice') return <NoticePage title="钱包通知" group="money" />;
  if (path === '/message/notice-setting') return <NoticeSettingPage />;
  if (matchesRoute(path, '/message') || matchesRoute(path, '/m/message')) return <MessagePage />;

  if (path === '/people/find-acquaintance') return <PeopleListPage title="发现朋友" desc="可能认识的人" />;
  if (path === '/people/follow-and-fans') return <MessagePeoplePage />;
  if (path === '/address-list') return <PeopleListPage title="通讯录朋友" desc="通讯录朋友" />;
  if (path === '/scan') return <ScanPage />;
  if (path === '/face-to-face') return <PeopleListPage title="面对面加朋友" desc="面对面加朋友" />;
  if (path === '/set-remark') return <RemarkPage />;

  if (matchesRoute(path, '/publish') || matchesRoute(path, '/m/publish')) return <PublishPage />;
  if (path === '/login/other') return <OtherLoginPage />;
  if (path === '/login/password') return <PasswordLoginPage />;
  if (path === '/login/verification-code') return <VerificationCodePage />;
  if (path === '/login/retrieve-password') return <RetrievePasswordPage />;
  if (path === '/login/help') return <LoginHelpPage />;
  if (matchesRoute(path, '/login')) return <LoginPage />;

  if (path === '/me/edit-userinfo') return <EditUserInfoPage />;
  if (path === '/me/edit-userinfo-item') return <EditUserInfoItemPage />;
  if (path === '/me/country-choose') return <CountryChoosePage />;
  if (path === '/me/my-card') return <MyCardPage />;
  if (path === '/me/add-school') return <AddSchoolPage />;
  if (path === '/me/choose-school') return <ChooseSchoolPage />;
  if (path === '/me/declare-school') return <DeclareSchoolPage />;
  if (path === '/me/choose-department') return <ChooseDepartmentPage />;
  if (path === '/me/display-type') return <DisplayTypePage />;
  if (path === '/me/choose-location') return <ChooseLocationPage />;
  if (path === '/me/choose-province') return <ChooseProvincePage />;
  if (path === '/me/choose-city') return <ChooseCityPage />;
  if (path === '/me/right-menu/look-history') return <LookHistoryPage title="观看历史" />;
  if (path === '/me/right-menu/minor-protection/index') return <MinorProtectionPage />;
  if (path === '/me/right-menu/minor-protection/detail-setting') return <MinorProtectionDetailPage />;
  if (path === '/me/right-menu/minor-protection/trigger-time') return <TriggerTimePage />;
  if (path === '/me/right-menu/setting') return <ProfileSettingReplicaPage />;
  if (path === '/me/collect/music-collect') return <MusicPage />;
  if (path === '/me/collect/video-collect') return <LookHistoryPage title="视频收藏" collect />;
  if (path === '/me/my-music') return <MusicPage />;
  if (path === '/me/request-update') return <RequestUpdatePage />;
  if (path === '/me/my-request-update') return <MyRequestUpdatePage />;
  if (matchesRoute(path, '/m/profile') || matchesRoute(path, '/me')) return <HomePage />;
  return <HomePage />;
}

export function Router() {
  const [path, setPath] = useState(location.pathname);

  useEffect(() => {
    const syncPath = () => setPath(location.pathname);
    window.addEventListener('popstate', syncPath);
    window.addEventListener(SPA_NAVIGATE_EVENT, syncPath);
    return () => {
      window.removeEventListener('popstate', syncPath);
      window.removeEventListener(SPA_NAVIGATE_EVENT, syncPath);
    };
  }, []);

  return (
    <Suspense fallback={<main className="mobileShell"><section className="emptyState">正在加载页面...</section></main>}>
      {routeForPath(path)}
    </Suspense>
  );
}

type DouyinLoadingProps = {
  mode?: 'full' | 'inline';
  type?: 'normal' | 'small';
  label?: string;
};

export function DouyinLoading({ mode = 'full', type = 'normal', label = '加载中' }: DouyinLoadingProps) {
  return (
    <div className={`dyLoading ${mode} ${type}`} role="status" aria-label={label}>
      <div className="dyLoadingCircle blue" />
      <div className="dyLoadingCircle red" />
    </div>
  );
}

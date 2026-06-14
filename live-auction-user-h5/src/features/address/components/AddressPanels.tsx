import { useMemo, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { Check, Circle, CircleHelp, Contact, Copy, MapPin, Pencil, Trash2, X } from 'lucide-react';
import type { DeliveryAddress, DeliveryAddressInput } from '../../../shared/address/addressBook';
import {
  formatAddressClipboardText,
  formatAddressRegion,
} from '../../../shared/address/addressBook';

type AddressListSheetProps = {
  addresses: DeliveryAddress[];
  selectedAddressId?: string;
  onSelect: (address: DeliveryAddress) => void;
  onAdd: () => void;
  onEdit?: (address: DeliveryAddress) => void;
  onClose: () => void;
  onSetDefault: (id: string) => void;
  onDelete: (id: string) => void;
};

type AddressFormProps = {
  title?: string;
  submitLabel?: string;
  embedded?: boolean;
  initialAddress?: DeliveryAddress;
  onSave: (input: DeliveryAddressInput) => void | Promise<void>;
};

type AddressFormSheetProps = AddressFormProps & {
  onClose: () => void;
};

type AddressField = 'receiver' | 'phone' | 'region' | 'detail';

type AddressCardProps = {
  address: DeliveryAddress;
  selected?: boolean;
  interactive?: boolean;
  onSelect?: (address: DeliveryAddress) => void;
  onEdit?: (address: DeliveryAddress) => void;
  onSetDefault: (id: string) => void;
  onDelete: (id: string) => void;
};

const EMPTY_ADDRESS: DeliveryAddressInput = {
  province: '',
  city: '',
  district: '',
  street: '',
  detail: '',
  receiver: '',
  phone: '',
  isDefault: false,
};

function copyAddress(address: DeliveryAddress) {
  if (typeof navigator === 'undefined' || !navigator.clipboard) return;
  void navigator.clipboard.writeText(formatAddressClipboardText(address)).catch(() => undefined);
}

export function DeliveryAddressCard({
  address,
  selected,
  interactive,
  onSelect,
  onEdit,
  onSetDefault,
  onDelete,
}: AddressCardProps) {
  const handleSelect = () => {
    if (interactive && onSelect) onSelect(address);
  };

  return (
    <article className={`deliveryAddressCard${selected ? ' isSelected' : ''}`}>
      <button className="deliveryAddressMain" type="button" onClick={handleSelect} disabled={!interactive}>
        <span>{formatAddressRegion(address)}</span>
        <b>{address.detail}</b>
        <small>{address.receiver} <em>{address.phone}</em></small>
      </button>
      {selected ? <Check className="deliveryAddressPicked" size={31} /> : null}
      <footer>
        <button
          className={address.isDefault ? 'isDefault' : ''}
          type="button"
          onClick={() => onSetDefault(address.id)}
        >
          {address.isDefault ? <Check size={18} /> : <Circle size={18} />}
          {address.isDefault ? '已设为默认' : '设为默认'}
        </button>
        <span />
        <button type="button" onClick={() => onDelete(address.id)}><Trash2 size={16} />删除</button>
        <button type="button" onClick={() => copyAddress(address)}><Copy size={16} />复制</button>
        {onEdit ? <button type="button" onClick={() => onEdit(address)}><Pencil size={16} />修改</button> : null}
      </footer>
    </article>
  );
}

export function DeliveryAddressListSheet({
  addresses,
  selectedAddressId,
  onSelect,
  onAdd,
  onEdit,
  onClose,
  onSetDefault,
  onDelete,
}: AddressListSheetProps) {
  return (
    <div className="deliveryAddressMask" role="presentation">
      <section className="deliveryAddressSheet" aria-modal="true" role="dialog" aria-label="收货地址">
        <header>
          <h2>收货地址 <CircleHelp size={17} /></h2>
          <button type="button" aria-label="关闭收货地址" onClick={onClose}><X size={28} /></button>
        </header>
        <div className="deliveryAddressList">
          {addresses.length > 0 ? addresses.map((address) => (
            <DeliveryAddressCard
              key={address.id}
              address={address}
              interactive
              selected={address.id === selectedAddressId}
              onSelect={onSelect}
              onEdit={onEdit}
              onSetDefault={onSetDefault}
              onDelete={onDelete}
            />
          )) : (
            <section className="deliveryAddressEmpty">
              <MapPin size={34} />
              <b>还没有收货地址</b>
              <span>新增一个地址后，就能在支付保证金时直接选择。</span>
            </section>
          )}
        </div>
        <div className="deliveryAddressSheetAction">
          <button type="button" onClick={onAdd}>+ 新建地址</button>
        </div>
      </section>
    </div>
  );
}

export function DeliveryAddressForm({ title, submitLabel = '保存', embedded = false, initialAddress, onSave }: AddressFormProps) {
  const [form, setForm] = useState<DeliveryAddressInput>(() => initialAddress ? {
    province: '',
    city: '',
    district: '',
    street: formatAddressRegion(initialAddress),
    detail: initialAddress.detail,
    receiver: initialAddress.receiver,
    phone: initialAddress.phone,
    isDefault: initialAddress.isDefault,
  } : EMPTY_ADDRESS);
  const [regionText, setRegionText] = useState(() => initialAddress ? formatAddressRegion(initialAddress) : '');
  const [activeErrorField, setActiveErrorField] = useState<AddressField | null>(null);
  const receiverRef = useRef<HTMLInputElement>(null);
  const phoneRef = useRef<HTMLInputElement>(null);
  const regionRef = useRef<HTMLInputElement>(null);
  const detailRef = useRef<HTMLInputElement>(null);

  const canSave = useMemo(() => (
    form.receiver.trim().length > 0 &&
    /^1\d{10}$/.test(form.phone.trim()) &&
    regionText.trim().length > 0 &&
    form.detail.trim().length > 0
  ), [form, regionText]);

  const update = (field: keyof DeliveryAddressInput, value: string | boolean) => {
    setForm((current) => ({ ...current, [field]: value }));
  };

  const firstInvalidField = (): AddressField | null => {
    if (!form.receiver.trim()) return 'receiver';
    if (!/^1\d{10}$/.test(form.phone.trim())) return 'phone';
    if (!regionText.trim()) return 'region';
    if (!form.detail.trim()) return 'detail';
    return null;
  };

  const focusField = (field: AddressField) => {
    const target = {
      receiver: receiverRef,
      phone: phoneRef,
      region: regionRef,
      detail: detailRef,
    }[field].current;
    window.setTimeout(() => target?.focus(), 0);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const invalidField = firstInvalidField();
    if (invalidField) {
      setActiveErrorField(invalidField);
      focusField(invalidField);
      return;
    }
    setActiveErrorField(null);
    onSave({
      ...form,
      province: form.province.trim(),
      city: form.city.trim(),
      district: form.district.trim(),
      street: regionText.trim(),
      detail: form.detail.trim(),
      receiver: form.receiver.trim(),
      phone: form.phone.trim(),
    });
  };

  return (
    <form className={`deliveryAddressForm${embedded ? ' isEmbedded' : ''}`} onSubmit={handleSubmit}>
      {title ? <h2>{title}</h2> : null}
      <label className="deliveryAddressPaste">
        <span>粘贴文本后将自动识别地址信息</span>
        <button type="button">粘贴并识别</button>
      </label>
      <section className="deliveryAddressFields">
        <label className={activeErrorField === 'receiver' && !form.receiver.trim() ? 'isInvalid' : ''}>
          <span>收货人</span>
          <input
            ref={receiverRef}
            value={form.receiver}
            onChange={(event) => update('receiver', event.target.value)}
            placeholder="请输入收货人姓名"
            autoComplete="name"
            aria-invalid={activeErrorField === 'receiver' && !form.receiver.trim()}
          />
          <em><Contact size={18} />通讯录</em>
          {activeErrorField === 'receiver' && !form.receiver.trim() ? <small>请填写收货人</small> : null}
        </label>
        <label className={activeErrorField === 'phone' && !/^1\d{10}$/.test(form.phone.trim()) ? 'isInvalid' : ''}>
          <span>手机号</span>
          <input
            ref={phoneRef}
            value={form.phone}
            onChange={(event) => update('phone', event.target.value.replace(/\D/g, '').slice(0, 11))}
            placeholder="请输入手机号"
            inputMode="tel"
            autoComplete="tel"
            aria-invalid={activeErrorField === 'phone' && !/^1\d{10}$/.test(form.phone.trim())}
          />
          {activeErrorField === 'phone' && !/^1\d{10}$/.test(form.phone.trim()) ? <small>请填写正确手机号</small> : null}
        </label>
        <label className={activeErrorField === 'region' && !regionText.trim() ? 'isInvalid' : ''}>
          <span>地区</span>
          <input
            ref={regionRef}
            value={regionText}
            onChange={(event) => setRegionText(event.target.value)}
            placeholder="选择省市区街道"
            aria-invalid={activeErrorField === 'region' && !regionText.trim()}
          />
          <em><MapPin size={18} />定位</em>
          {activeErrorField === 'region' && !regionText.trim() ? <small>请选择省市区街道</small> : null}
        </label>
        <label className={activeErrorField === 'detail' && !form.detail.trim() ? 'isInvalid' : ''}>
          <span>详细地址</span>
          <input
            ref={detailRef}
            value={form.detail}
            onChange={(event) => update('detail', event.target.value)}
            placeholder="小区楼栋、门牌号、村等"
            aria-invalid={activeErrorField === 'detail' && !form.detail.trim()}
          />
          {activeErrorField === 'detail' && !form.detail.trim() ? <small>请填写详细地址</small> : null}
        </label>
      </section>
      <label className="deliveryAddressDefaultSwitch">
        <span>默认地址</span>
        <input
          type="checkbox"
          checked={form.isDefault}
          onChange={(event) => update('isDefault', event.target.checked)}
        />
      </label>
      <button className={`deliveryAddressSave${canSave ? '' : ' isIncomplete'}`} type="submit" aria-disabled={!canSave}>
        {submitLabel}
      </button>
    </form>
  );
}

export function DeliveryAddressFormSheet({ title = '新建地址', submitLabel, initialAddress, onSave, onClose }: AddressFormSheetProps) {
  return (
    <div className="deliveryAddressMask" role="presentation">
      <section className="deliveryAddressSheet isForm" aria-modal="true" role="dialog" aria-label={title}>
        <header>
          <h2>{title}</h2>
          <button type="button" aria-label="关闭新建地址" onClick={onClose}><X size={28} /></button>
        </header>
        <DeliveryAddressForm submitLabel={submitLabel} initialAddress={initialAddress} onSave={onSave} />
      </section>
    </div>
  );
}

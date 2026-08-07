/* Shared UI atoms — faithful MUI re-implementations of the prototype's helper
   render functions (icon, avatar, chip, section title, buttons, form fields). */
import React, { useEffect, useState } from 'react';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import AddCircleOutlinedIcon from '@mui/icons-material/AddCircleOutlined';
import AddPhotoAlternateOutlinedIcon from '@mui/icons-material/AddPhotoAlternateOutlined';
import AdminPanelSettingsOutlinedIcon from '@mui/icons-material/AdminPanelSettingsOutlined';
import ArchiveOutlinedIcon from '@mui/icons-material/ArchiveOutlined';
import UnarchiveOutlinedIcon from '@mui/icons-material/UnarchiveOutlined';
import ArrowBackOutlinedIcon from '@mui/icons-material/ArrowBackOutlined';
import AppsOutlinedIcon from '@mui/icons-material/AppsOutlined';
import AutorenewOutlinedIcon from '@mui/icons-material/AutorenewOutlined';
import BadgeOutlinedIcon from '@mui/icons-material/BadgeOutlined';
import BeachAccessOutlinedIcon from '@mui/icons-material/BeachAccessOutlined';
import BlockOutlinedIcon from '@mui/icons-material/BlockOutlined';
import BrightnessAutoOutlinedIcon from '@mui/icons-material/BrightnessAutoOutlined';
import CakeOutlinedIcon from '@mui/icons-material/CakeOutlined';
import CalendarMonthOutlinedIcon from '@mui/icons-material/CalendarMonthOutlined';
import CampaignOutlinedIcon from '@mui/icons-material/CampaignOutlined';
import CancelOutlinedIcon from '@mui/icons-material/CancelOutlined';
import CategoryOutlinedIcon from '@mui/icons-material/CategoryOutlined';
import CelebrationOutlinedIcon from '@mui/icons-material/CelebrationOutlined';
import ChatBubbleOutlinedIcon from '@mui/icons-material/ChatBubbleOutlined';
import CheckOutlinedIcon from '@mui/icons-material/CheckOutlined';
import CheckBoxOutlinedIcon from '@mui/icons-material/CheckBoxOutlined';
import CheckBoxOutlineBlankOutlinedIcon from '@mui/icons-material/CheckBoxOutlineBlankOutlined';
import CheckCircleOutlinedIcon from '@mui/icons-material/CheckCircleOutlined';
import ChevronLeftOutlinedIcon from '@mui/icons-material/ChevronLeftOutlined';
import ChevronRightOutlinedIcon from '@mui/icons-material/ChevronRightOutlined';
import CircleOutlinedIcon from '@mui/icons-material/CircleOutlined';
import CloseOutlinedIcon from '@mui/icons-material/CloseOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import DarkModeOutlinedIcon from '@mui/icons-material/DarkModeOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import DeleteForeverOutlinedIcon from '@mui/icons-material/DeleteForeverOutlined';
import DownloadOutlinedIcon from '@mui/icons-material/DownloadOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import EmojiEventsOutlinedIcon from '@mui/icons-material/EmojiEventsOutlined';
import ErrorOutlinedIcon from '@mui/icons-material/ErrorOutlined';
import EventOutlinedIcon from '@mui/icons-material/EventOutlined';
import EventAvailableOutlinedIcon from '@mui/icons-material/EventAvailableOutlined';
import EventBusyOutlinedIcon from '@mui/icons-material/EventBusyOutlined';
import ExpandLessOutlinedIcon from '@mui/icons-material/ExpandLessOutlined';
import ExpandMoreOutlinedIcon from '@mui/icons-material/ExpandMoreOutlined';
import FitnessCenterOutlinedIcon from '@mui/icons-material/FitnessCenterOutlined';
import GavelOutlinedIcon from '@mui/icons-material/GavelOutlined';
import GridOnOutlinedIcon from '@mui/icons-material/GridOnOutlined';
import GroupOutlinedIcon from '@mui/icons-material/GroupOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import HelpOutlinedIcon from '@mui/icons-material/HelpOutlined';
import HomeOutlinedIcon from '@mui/icons-material/HomeOutlined';
import HourglassTopOutlinedIcon from '@mui/icons-material/HourglassTopOutlined';
import HowToVoteOutlinedIcon from '@mui/icons-material/HowToVoteOutlined';
import IncompleteCircleOutlinedIcon from '@mui/icons-material/IncompleteCircleOutlined';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import InsightsOutlinedIcon from '@mui/icons-material/InsightsOutlined';
import InstallMobileOutlinedIcon from '@mui/icons-material/InstallMobileOutlined';
import IosShareOutlinedIcon from '@mui/icons-material/IosShareOutlined';
import LightModeOutlinedIcon from '@mui/icons-material/LightModeOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import LogoutOutlinedIcon from '@mui/icons-material/LogoutOutlined';
import MailOutlinedIcon from '@mui/icons-material/MailOutlined';
import MenuBookOutlinedIcon from '@mui/icons-material/MenuBookOutlined';
import MusicNoteOutlinedIcon from '@mui/icons-material/MusicNoteOutlined';
import NoteOutlinedIcon from '@mui/icons-material/NoteOutlined';
import NotificationsOutlinedIcon from '@mui/icons-material/NotificationsOutlined';
import NotificationsOffOutlinedIcon from '@mui/icons-material/NotificationsOffOutlined';
import PaymentsOutlinedIcon from '@mui/icons-material/PaymentsOutlined';
import PaletteOutlinedIcon from '@mui/icons-material/PaletteOutlined';
import PendingActionsOutlinedIcon from '@mui/icons-material/PendingActionsOutlined';
import PersonOutlinedIcon from '@mui/icons-material/PersonOutlined';
import PersonAddOutlinedIcon from '@mui/icons-material/PersonAddOutlined';
import PersonOffOutlinedIcon from '@mui/icons-material/PersonOffOutlined';
import PersonRemoveOutlinedIcon from '@mui/icons-material/PersonRemoveOutlined';
import PhotoCameraOutlinedIcon from '@mui/icons-material/PhotoCameraOutlined';
import PhoneIphoneOutlinedIcon from '@mui/icons-material/PhoneIphoneOutlined';
import PlaceOutlinedIcon from '@mui/icons-material/PlaceOutlined';
import PushPinOutlinedIcon from '@mui/icons-material/PushPinOutlined';
import PrivacyTipOutlinedIcon from '@mui/icons-material/PrivacyTipOutlined';
import QuestionMarkOutlinedIcon from '@mui/icons-material/QuestionMarkOutlined';
import ReceiptLongOutlinedIcon from '@mui/icons-material/ReceiptLongOutlined';
import RemoveOutlinedIcon from '@mui/icons-material/RemoveOutlined';
import RepeatOutlinedIcon from '@mui/icons-material/RepeatOutlined';
import SavingsOutlinedIcon from '@mui/icons-material/SavingsOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import SearchOffOutlinedIcon from '@mui/icons-material/SearchOffOutlined';
import SendOutlinedIcon from '@mui/icons-material/SendOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import ShieldOutlinedIcon from '@mui/icons-material/ShieldOutlined';
import StickyNote2OutlinedIcon from '@mui/icons-material/StickyNote2Outlined';
import SyncOutlinedIcon from '@mui/icons-material/SyncOutlined';
import TitleOutlinedIcon from '@mui/icons-material/TitleOutlined';
import ToggleOffOutlinedIcon from '@mui/icons-material/ToggleOffOutlined';
import ToggleOnOutlinedIcon from '@mui/icons-material/ToggleOnOutlined';
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined';
import UnfoldMoreOutlinedIcon from '@mui/icons-material/UnfoldMoreOutlined';
import WarningOutlinedIcon from '@mui/icons-material/WarningOutlined';
import ImageOutlinedIcon from '@mui/icons-material/ImageOutlined';
import NorthEastOutlinedIcon from '@mui/icons-material/NorthEastOutlined';
import SouthWestOutlinedIcon from '@mui/icons-material/SouthWestOutlined';
import SportsOutlinedIcon from '@mui/icons-material/SportsOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined';
import type { SvgIconComponent } from '@mui/icons-material';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import MuiSkeleton from '@mui/material/Skeleton';
import { type SxProps, type Theme } from '@mui/material/styles';
import { initials as toInitials, NEUTRAL } from '@/styles/tokens';
import { useApp } from '@/context/AppContext';
import { t } from '@/i18n';

const ICONS: Record<string, SvgIconComponent> = {
  add: AddOutlinedIcon,
  add_circle: AddCircleOutlinedIcon,
  add_photo_alternate: AddPhotoAlternateOutlinedIcon,
  admin_panel_settings: AdminPanelSettingsOutlinedIcon,
  archive: ArchiveOutlinedIcon,
  unarchive: UnarchiveOutlinedIcon,
  arrow_back: ArrowBackOutlinedIcon,
  apps: AppsOutlinedIcon,
  autorenew: AutorenewOutlinedIcon,
  badge: BadgeOutlinedIcon,
  beach_access: BeachAccessOutlinedIcon,
  block: BlockOutlinedIcon,
  brightness_auto: BrightnessAutoOutlinedIcon,
  cake: CakeOutlinedIcon,
  calendar_month: CalendarMonthOutlinedIcon,
  campaign: CampaignOutlinedIcon,
  cancel: CancelOutlinedIcon,
  category: CategoryOutlinedIcon,
  celebration: CelebrationOutlinedIcon,
  chat_bubble: ChatBubbleOutlinedIcon,
  check: CheckOutlinedIcon,
  check_box: CheckBoxOutlinedIcon,
  check_box_outline_blank: CheckBoxOutlineBlankOutlinedIcon,
  check_circle: CheckCircleOutlinedIcon,
  chevron_left: ChevronLeftOutlinedIcon,
  chevron_right: ChevronRightOutlinedIcon,
  circle: CircleOutlinedIcon,
  close: CloseOutlinedIcon,
  content_copy: ContentCopyOutlinedIcon,
  dark_mode: DarkModeOutlinedIcon,
  delete: DeleteOutlinedIcon,
  delete_forever: DeleteForeverOutlinedIcon,
  download: DownloadOutlinedIcon,
  edit: EditOutlinedIcon,
  emoji_events: EmojiEventsOutlinedIcon,
  error: ErrorOutlinedIcon,
  event: EventOutlinedIcon,
  event_available: EventAvailableOutlinedIcon,
  event_busy: EventBusyOutlinedIcon,
  expand_less: ExpandLessOutlinedIcon,
  expand_more: ExpandMoreOutlinedIcon,
  fitness_center: FitnessCenterOutlinedIcon,
  gavel: GavelOutlinedIcon,
  grid_on: GridOnOutlinedIcon,
  group: GroupOutlinedIcon,
  groups: GroupsOutlinedIcon,
  help: HelpOutlinedIcon,
  home: HomeOutlinedIcon,
  hourglass_top: HourglassTopOutlinedIcon,
  how_to_vote: HowToVoteOutlinedIcon,
  incomplete_circle: IncompleteCircleOutlinedIcon,
  info: InfoOutlinedIcon,
  insights: InsightsOutlinedIcon,
  install_mobile: InstallMobileOutlinedIcon,
  ios_share: IosShareOutlinedIcon,
  light_mode: LightModeOutlinedIcon,
  link: LinkOutlinedIcon,
  lock: LockOutlinedIcon,
  login: LoginOutlinedIcon,
  logout: LogoutOutlinedIcon,
  mail: MailOutlinedIcon,
  menu_book: MenuBookOutlinedIcon,
  music_note: MusicNoteOutlinedIcon,
  note: NoteOutlinedIcon,
  notifications: NotificationsOutlinedIcon,
  notifications_off: NotificationsOffOutlinedIcon,
  payments: PaymentsOutlinedIcon,
  palette: PaletteOutlinedIcon,
  pending_actions: PendingActionsOutlinedIcon,
  person: PersonOutlinedIcon,
  person_add: PersonAddOutlinedIcon,
  person_off: PersonOffOutlinedIcon,
  person_remove: PersonRemoveOutlinedIcon,
  photo_camera: PhotoCameraOutlinedIcon,
  phone_iphone: PhoneIphoneOutlinedIcon,
  place: PlaceOutlinedIcon,
  push_pin: PushPinOutlinedIcon,
  privacy_tip: PrivacyTipOutlinedIcon,
  question_mark: QuestionMarkOutlinedIcon,
  receipt_long: ReceiptLongOutlinedIcon,
  remove: RemoveOutlinedIcon,
  repeat: RepeatOutlinedIcon,
  savings: SavingsOutlinedIcon,
  schedule: ScheduleOutlinedIcon,
  search_off: SearchOffOutlinedIcon,
  send: SendOutlinedIcon,
  settings: SettingsOutlinedIcon,
  shield: ShieldOutlinedIcon,
  shield_person: AdminPanelSettingsOutlinedIcon,
  sticky_note_2: StickyNote2OutlinedIcon,
  sync: SyncOutlinedIcon,
  title: TitleOutlinedIcon,
  toggle_off: ToggleOffOutlinedIcon,
  toggle_on: ToggleOnOutlinedIcon,
  tune: TuneOutlinedIcon,
  unfold_more: UnfoldMoreOutlinedIcon,
  warning: WarningOutlinedIcon,
  image: ImageOutlinedIcon,
  north_east: NorthEastOutlinedIcon,
  south_west: SouthWestOutlinedIcon,
  sports: SportsOutlinedIcon,
  event_upcoming: EventAvailableOutlinedIcon,
  visibility: VisibilityOutlinedIcon,
  visibility_off: VisibilityOffOutlinedIcon,
};

/** SVG icon rendered by symbolic name.
 *  Decorative by default (aria-hidden); pass `label` to expose it to assistive
 *  tech as a standalone meaningful icon. */
export function Sym({
  name,
  size = 20,
  color = 'inherit',
  sx,
  label,
}: {
  name: string;
  size?: number;
  color?: string;
  sx?: SxProps<Theme>;
  label?: string;
}) {
  const Icon = ICONS[name] ?? HelpOutlinedIcon;
  return (
    <Icon
      aria-hidden={label ? undefined : true}
      role={label ? 'img' : undefined}
      aria-label={label}
      sx={{
        fontSize: size + 'px',
        color,
        flex: '0 0 auto',
        ...(sx as object),
      }}
    />
  );
}

/** Round avatar with photo or coloured initials. */
export function Av({
  name,
  photo,
  color,
  size = 40,
  font,
}: {
  // Explicit `| undefined` throughout -- callers pass these straight from
  // API-mapped rows (`row.photo`, `row.color`, ...) where "no photo/color/
  // name" is `undefined`, not a distinct value from "prop omitted".
  name?: string | undefined;
  photo?: string | null | undefined;
  color?: string | undefined;
  size?: number;
  font?: number;
}) {
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    setFailed(false);
    if (!photo) return;
    const img = new Image();
    img.onerror = () => setFailed(true);
    img.src = photo;
    return () => {
      img.onerror = null;
    };
  }, [photo]);

  const f = font || Math.round(size * 0.36);
  const base: SxProps<Theme> = {
    width: size,
    height: size,
    borderRadius: '50%',
    flex: '0 0 auto',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#fff',
    fontWeight: 700,
    fontSize: f + 'px',
    overflow: 'hidden',
    backgroundColor: color || '#888',
  };
  if (photo && !failed) {
    return (
      <Box
        component="span"
        role="img"
        aria-label={name || ''}
        sx={{
          ...(base as object),
          backgroundImage: `url(${photo})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
        }}
      />
    );
  }
  return (
    <Box component="span" sx={base}>
      {toInitials(name || '')}
    </Box>
  );
}

/** Pill chip (status / type / label). */
export function Chip({
  label,
  color,
  bg,
  icon,
  fs = 11,
}: {
  label: React.ReactNode;
  color: string;
  bg: string;
  icon?: string;
  fs?: number;
}) {
  return (
    <Box
      component="span"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        fontSize: fs + 'px',
        fontWeight: 700,
        color,
        background: bg,
        padding: '4px 9px',
        borderRadius: '999px',
        whiteSpace: 'nowrap',
      }}
    >
      {icon ? <Sym name={icon} size={fs + 3} color={color} /> : null}
      {label}
    </Box>
  );
}

export function SectionTitle({ children, right }: { children: React.ReactNode; right?: React.ReactNode }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: '8px', m: '4px 2px 10px' }}>
      <Box
        sx={{
          fontSize: '12px',
          fontWeight: 700,
          color: NEUTRAL.secondary,
          letterSpacing: '.4px',
          textTransform: 'uppercase',
          flex: 1,
        }}
      >
        {children}
      </Box>
      {right || null}
    </Box>
  );
}

export function EmptyState({ icon, text }: { icon: string; text: string }) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '10px',
        p: '56px 20px',
        color: NEUTRAL.faint,
      }}
    >
      <Sym name={icon} size={46} />
      <Box sx={{ fontSize: '14px', textAlign: 'center' }}>{text}</Box>
    </Box>
  );
}

export function SpinnerBox() {
  const { state } = useApp();
  const { primaryColor } = state;
  return (
    <Box role="status" aria-label={t('common.loading')} sx={{ display: 'flex', justifyContent: 'center', p: '48px' }}>
      <Box
        sx={{
          width: 34,
          height: 34,
          border: `3px solid ${NEUTRAL.line3}`,
          borderTopColor: primaryColor,
          borderRadius: '50%',
          animation: 'tvSpin .8s linear infinite',
        }}
      />
    </Box>
  );
}

/** Skeleton placeholder for list/card views while data loads. */
export function SkeletonList({ rows = 4, rowHeight = 80 }: { rows?: number; rowHeight?: number }) {
  return (
    <Box
      role="status"
      aria-label={t('common.loading')}
      sx={{ display: 'flex', flexDirection: 'column', gap: '10px', maxWidth: '820px' }}
    >
      {Array.from({ length: rows }, (_, i) => (
        <MuiSkeleton key={i} variant="rounded" height={rowHeight} sx={{ borderRadius: '16px' }} />
      ))}
    </Box>
  );
}

export function Spinner({ size = 16, color = 'currentColor' }: { size?: number; color?: string }) {
  return (
    <Box
      component="span"
      role="status"
      aria-label={t('common.loading')}
      sx={{
        width: size,
        height: size,
        border: `2px solid ${color}`,
        borderTopColor: 'transparent',
        borderRadius: '50%',
        display: 'inline-block',
        animation: 'tvSpin .7s linear infinite',
      }}
    />
  );
}

export function Card({ children, sx }: { children: React.ReactNode; sx?: SxProps<Theme> }) {
  return (
    <Box
      sx={{
        background: NEUTRAL.card,
        border: `1px solid ${NEUTRAL.line}`,
        borderRadius: '18px',
        p: '16px',
        ...(sx as object),
      }}
    >
      {children}
    </Box>
  );
}

/** Full-width primary button matching the prototype's primaryBtn(). */
export function PrimaryButton({
  label,
  onClick,
  disabled,
  busy,
}: {
  label: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  busy?: boolean;
}) {
  const theme = useApp().state.primaryColor;
  return (
    <ButtonBase
      onClick={onClick}
      disabled={disabled || busy}
      aria-busy={busy ? true : undefined}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '9px',
        width: '100%',
        p: '14px',
        borderRadius: '14px',
        border: 'none',
        cursor: disabled ? 'default' : 'pointer',
        background: disabled || busy ? NEUTRAL.inputBorder : theme,
        color: '#fff',
        fontSize: '15px',
        fontWeight: 600,
        mt: '4px',
      }}
    >
      {busy ? <Spinner /> : null}
      {label}
    </ButtonBase>
  );
}

export const labelSx: SxProps<Theme> = {
  fontSize: '12px',
  fontWeight: 600,
  color: NEUTRAL.onSurfaceVariant,
  mb: '6px',
};
export const inputSx: React.CSSProperties = {
  width: '100%',
  border: `1.5px solid ${NEUTRAL.inputBorder}`,
  borderRadius: '13px',
  padding: '12px 14px',
  fontSize: '14px',
  outline: 'none',
  background: NEUTRAL.card,
  color: NEUTRAL.onSurface,
  fontFamily: 'inherit',
};

export function Field({
  label,
  children,
  required,
  error,
  errorText,
  helperText,
}: {
  label: string;
  children: React.ReactNode;
  required?: boolean;
  error?: boolean;
  // Explicit `| undefined` (not just `errorText?:`) -- callers pass
  // `errors.field?.message` straight from react-hook-form's FieldErrors,
  // which is `string | undefined` when there's no validation error, and
  // there's no meaningful difference here between "no errorText prop" and
  // "errorText is undefined" (both render no error message below).
  errorText?: string | undefined;
  helperText?: string;
}) {
  const errorId = errorText ? `field-err-${label.replace(/\s+/g, '-').toLowerCase()}` : undefined;
  return (
    <Box component="label" sx={{ display: 'block' }}>
      <Box sx={labelSx}>
        {label}
        {required ? (
          <Box component="span" aria-hidden="true" sx={{ color: NEUTRAL.error, ml: '2px' }}>
            *
          </Box>
        ) : null}
      </Box>
      {React.isValidElement(children)
        ? React.cloneElement(children as React.ReactElement<Record<string, unknown>>, {
            'aria-required': required ? 'true' : undefined,
            'aria-invalid': error ? 'true' : undefined,
            'aria-describedby': errorId,
            style: error
              ? {
                  ...(children as React.ReactElement<{ style?: React.CSSProperties }>).props.style,
                  borderColor: NEUTRAL.error,
                }
              : (children as React.ReactElement<{ style?: React.CSSProperties }>).props.style,
          })
        : children}
      {errorId && errorText ? (
        <Box id={errorId} role="alert" sx={{ fontSize: '12px', color: NEUTRAL.error, mt: '4px' }}>
          {errorText}
        </Box>
      ) : helperText ? (
        <Box sx={{ fontSize: '12px', color: NEUTRAL.secondary, mt: '4px' }}>{helperText}</Box>
      ) : null}
    </Box>
  );
}

type TextInputProps = React.InputHTMLAttributes<HTMLInputElement> & { name: string };

/** Form-bound text input (mirrors prototype tf()). */
export const TextInput = React.forwardRef<HTMLInputElement, TextInputProps>(
  ({ name, type = 'text', placeholder, min, max, style: styleProp, value, onChange, onBlur, ...rest }, ref) => (
    <input
      ref={ref}
      name={name}
      type={type}
      min={min}
      max={max}
      value={value == null ? undefined : String(value)}
      placeholder={placeholder || ''}
      onChange={onChange}
      onBlur={onBlur}
      style={styleProp ? { ...inputSx, ...styleProp } : inputSx}
      {...rest}
    />
  ),
);
TextInput.displayName = 'TextInput';

type TextAreaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
  name: string;
  minHeight?: number;
};

export const TextArea = React.forwardRef<HTMLTextAreaElement, TextAreaProps>(
  ({ name, placeholder, minHeight = 80, onBlur, maxLength, style: styleProp, value, onChange, ...rest }, ref) => (
    <textarea
      ref={ref}
      name={name}
      value={value == null ? undefined : String(value)}
      placeholder={placeholder || ''}
      onChange={onChange}
      onBlur={onBlur}
      maxLength={maxLength}
      style={
        styleProp
          ? { ...inputSx, minHeight, resize: 'vertical', ...styleProp }
          : { ...inputSx, minHeight, resize: 'vertical' }
      }
      {...rest}
    />
  ),
);
TextArea.displayName = 'TextArea';

/** Small square icon button used in lists. */
export function IconBtn({
  icon,
  onClick,
  color = NEUTRAL.secondary,
  bg = NEUTRAL.sidebar,
  title,
  size = 30,
}: {
  icon: string;
  onClick?: () => void;
  color?: string;
  bg?: string;
  title?: string;
  size?: number;
}) {
  return (
    <ButtonBase
      title={title}
      aria-label={title}
      onClick={onClick}
      sx={{ width: size, height: size, borderRadius: '8px', background: bg, color, flex: '0 0 auto' }}
    >
      <Sym name={icon} size={17} color={color} />
    </ButtonBase>
  );
}

export function metaItem(icon: string, text: React.ReactNode, key?: string) {
  return (
    <Box
      key={key}
      component="span"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        whiteSpace: 'nowrap',
        maxWidth: '180px',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      }}
    >
      <Sym name={icon} size={15} />
      {text}
    </Box>
  );
}

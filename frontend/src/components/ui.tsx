/* Shared UI atoms — faithful MUI re-implementations of the prototype's helper
   render functions (icon, avatar, chip, section title, buttons, form fields). */
import React, { useEffect, useState } from 'react';
import AddIcon from '@mui/icons-material/Add';
import AddCircleIcon from '@mui/icons-material/AddCircle';
import AddPhotoAlternateIcon from '@mui/icons-material/AddPhotoAlternate';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import BadgeIcon from '@mui/icons-material/Badge';
import BeachAccessIcon from '@mui/icons-material/BeachAccess';
import BlockIcon from '@mui/icons-material/Block';
import BrightnessAutoIcon from '@mui/icons-material/BrightnessAuto';
import CakeIcon from '@mui/icons-material/Cake';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import CampaignIcon from '@mui/icons-material/Campaign';
import CancelIcon from '@mui/icons-material/Cancel';
import CategoryIcon from '@mui/icons-material/Category';
import ChatBubbleIcon from '@mui/icons-material/ChatBubble';
import CheckIcon from '@mui/icons-material/Check';
import CheckBoxIcon from '@mui/icons-material/CheckBox';
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import CircleIcon from '@mui/icons-material/Circle';
import CloseIcon from '@mui/icons-material/Close';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import DeleteIcon from '@mui/icons-material/Delete';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import DownloadIcon from '@mui/icons-material/Download';
import EditIcon from '@mui/icons-material/Edit';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import ErrorIcon from '@mui/icons-material/Error';
import EventIcon from '@mui/icons-material/Event';
import EventAvailableIcon from '@mui/icons-material/EventAvailable';
import EventBusyIcon from '@mui/icons-material/EventBusy';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import GavelIcon from '@mui/icons-material/Gavel';
import GridOnIcon from '@mui/icons-material/GridOn';
import GroupsIcon from '@mui/icons-material/Groups';
import HelpIcon from '@mui/icons-material/Help';
import HelpOutlineIcon from '@mui/icons-material/HelpOutlined';
import HomeIcon from '@mui/icons-material/Home';
import HourglassTopIcon from '@mui/icons-material/HourglassTop';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import InfoIcon from '@mui/icons-material/Info';
import InsightsIcon from '@mui/icons-material/Insights';
import InstallMobileIcon from '@mui/icons-material/InstallMobile';
import IosShareIcon from '@mui/icons-material/IosShare';
import LightModeIcon from '@mui/icons-material/LightMode';
import LinkIcon from '@mui/icons-material/Link';
import LockIcon from '@mui/icons-material/Lock';
import LoginIcon from '@mui/icons-material/Login';
import LogoutIcon from '@mui/icons-material/Logout';
import MailIcon from '@mui/icons-material/Mail';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import NoteIcon from '@mui/icons-material/Note';
import NotificationsIcon from '@mui/icons-material/Notifications';
import NotificationsOffIcon from '@mui/icons-material/NotificationsOff';
import PaymentsIcon from '@mui/icons-material/Payments';
import PendingActionsIcon from '@mui/icons-material/PendingActions';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import PersonOffIcon from '@mui/icons-material/PersonOff';
import PersonRemoveIcon from '@mui/icons-material/PersonRemove';
import PhotoCameraIcon from '@mui/icons-material/PhotoCamera';
import PhoneIphoneIcon from '@mui/icons-material/PhoneIphone';
import PlaceIcon from '@mui/icons-material/Place';
import PushPinIcon from '@mui/icons-material/PushPin';
import QuestionMarkIcon from '@mui/icons-material/QuestionMark';
import ReceiptLongIcon from '@mui/icons-material/ReceiptLong';
import RemoveIcon from '@mui/icons-material/Remove';
import RepeatIcon from '@mui/icons-material/Repeat';
import SavingsIcon from '@mui/icons-material/Savings';
import ScheduleIcon from '@mui/icons-material/Schedule';
import SearchOffIcon from '@mui/icons-material/SearchOff';
import SendIcon from '@mui/icons-material/Send';
import SettingsIcon from '@mui/icons-material/Settings';
import StickyNote2Icon from '@mui/icons-material/StickyNote2';
import SyncIcon from '@mui/icons-material/Sync';
import TitleIcon from '@mui/icons-material/Title';
import ToggleOffIcon from '@mui/icons-material/ToggleOff';
import ToggleOnIcon from '@mui/icons-material/ToggleOn';
import TuneIcon from '@mui/icons-material/Tune';
import UnfoldMoreIcon from '@mui/icons-material/UnfoldMore';
import WarningIcon from '@mui/icons-material/Warning';
import ImageIcon from '@mui/icons-material/Image';
import NorthEastIcon from '@mui/icons-material/NorthEast';
import SouthWestIcon from '@mui/icons-material/SouthWest';
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import type { SvgIconComponent } from '@mui/icons-material';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import MuiSkeleton from '@mui/material/Skeleton';
import { type SxProps, type Theme } from '@mui/material/styles';
import { initials as toInitials, NEUTRAL } from '@/styles/tokens';
import { useApp } from '@/context/AppContext';
import { t } from '@/i18n';

const ICONS: Record<string, SvgIconComponent> = {
  add: AddIcon,
  add_circle: AddCircleIcon,
  add_photo_alternate: AddPhotoAlternateIcon,
  admin_panel_settings: AdminPanelSettingsIcon,
  arrow_back: ArrowBackIcon,
  autorenew: AutorenewIcon,
  badge: BadgeIcon,
  beach_access: BeachAccessIcon,
  block: BlockIcon,
  brightness_auto: BrightnessAutoIcon,
  cake: CakeIcon,
  calendar_month: CalendarMonthIcon,
  campaign: CampaignIcon,
  cancel: CancelIcon,
  category: CategoryIcon,
  chat_bubble: ChatBubbleIcon,
  check: CheckIcon,
  check_box: CheckBoxIcon,
  check_box_outline_blank: CheckBoxOutlineBlankIcon,
  check_circle: CheckCircleIcon,
  chevron_left: ChevronLeftIcon,
  chevron_right: ChevronRightIcon,
  circle: CircleIcon,
  close: CloseIcon,
  content_copy: ContentCopyIcon,
  dark_mode: DarkModeIcon,
  delete: DeleteIcon,
  delete_forever: DeleteForeverIcon,
  download: DownloadIcon,
  edit: EditIcon,
  emoji_events: EmojiEventsIcon,
  error: ErrorIcon,
  event: EventIcon,
  event_available: EventAvailableIcon,
  event_busy: EventBusyIcon,
  expand_less: ExpandLessIcon,
  expand_more: ExpandMoreIcon,
  gavel: GavelIcon,
  grid_on: GridOnIcon,
  groups: GroupsIcon,
  help: HelpIcon,
  home: HomeIcon,
  hourglass_top: HourglassTopIcon,
  how_to_vote: HowToVoteIcon,
  info: InfoIcon,
  insights: InsightsIcon,
  install_mobile: InstallMobileIcon,
  ios_share: IosShareIcon,
  light_mode: LightModeIcon,
  link: LinkIcon,
  lock: LockIcon,
  login: LoginIcon,
  logout: LogoutIcon,
  mail: MailIcon,
  menu_book: MenuBookIcon,
  note: NoteIcon,
  notifications: NotificationsIcon,
  notifications_off: NotificationsOffIcon,
  payments: PaymentsIcon,
  pending_actions: PendingActionsIcon,
  person_add: PersonAddIcon,
  person_off: PersonOffIcon,
  person_remove: PersonRemoveIcon,
  photo_camera: PhotoCameraIcon,
  phone_iphone: PhoneIphoneIcon,
  place: PlaceIcon,
  push_pin: PushPinIcon,
  question_mark: QuestionMarkIcon,
  receipt_long: ReceiptLongIcon,
  remove: RemoveIcon,
  repeat: RepeatIcon,
  savings: SavingsIcon,
  schedule: ScheduleIcon,
  search_off: SearchOffIcon,
  send: SendIcon,
  settings: SettingsIcon,
  shield_person: AdminPanelSettingsIcon,
  sticky_note_2: StickyNote2Icon,
  sync: SyncIcon,
  title: TitleIcon,
  toggle_off: ToggleOffIcon,
  toggle_on: ToggleOnIcon,
  tune: TuneIcon,
  unfold_more: UnfoldMoreIcon,
  warning: WarningIcon,
  image: ImageIcon,
  north_east: NorthEastIcon,
  south_west: SouthWestIcon,
  event_upcoming: EventAvailableIcon,
  visibility: VisibilityIcon,
  visibility_off: VisibilityOffIcon,
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
  const Icon = ICONS[name] ?? HelpOutlineIcon;
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

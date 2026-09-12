import { IconChevronRight } from "@tabler/icons-react"
import {
  IconAtom,
  IconBrain,
  IconChartPie,
  IconChevronsDown,
  IconChevronsUp,
  IconClock,
  IconFileText,
  IconFlask,
  IconKey,
  IconListDetails,
  IconMessageCircle,
  IconSettings,
  IconSparkles,
  IconTools,
  IconTrendingUp,
} from "@tabler/icons-react"
import { Link, useRouterState } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import { AlertDot } from "@/components/ui/alert-dot"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useSandboxStatus } from "@/hooks/use-sandbox-status"
import { useSidebarChannels } from "@/hooks/use-sidebar-channels"

interface NavItem {
  title: string
  url: string
  icon: React.ComponentType<{ className?: string }>
  translateTitle?: boolean
  badge?: string
}

interface NavGroup {
  label: string
  defaultOpen: boolean
  items: NavItem[]
  isChannelsGroup?: boolean
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const routerState = useRouterState()
  const { t } = useTranslation()
  const currentPath = routerState.location.pathname
  const {
    channelItems,
    hasMoreChannels,
    showAllChannels,
    toggleShowAllChannels,
  } = useSidebarChannels({ t })
  const { data: sandboxStatus } = useSandboxStatus()

  const navGroups: NavGroup[] = React.useMemo(() => {
    const servicesItems: NavItem[] = [
      {
        title: "navigation.config",
        url: "/config",
        icon: IconSettings,
        translateTitle: true,
      },
      {
        title: "navigation.logs",
        url: "/logs",
        icon: IconListDetails,
        translateTitle: true,
      },
    ]

    // Add sandbox menu only if enabled
    if (sandboxStatus?.enabled) {
      servicesItems.push({
        title: "navigation.sandbox",
        url: "/sandbox",
        icon: IconFlask,
        translateTitle: true,
      })
    }

    const advancedItems: NavItem[] = [
      {
        title: "navigation.models",
        url: "/models",
        icon: IconAtom,
        translateTitle: true,
      },
      {
        title: "navigation.credentials",
        url: "/credentials",
        icon: IconKey,
        translateTitle: true,
      },
      ...channelItems.map((item) => ({
        title: item.title,
        url: item.url,
        icon: item.icon,
        translateTitle: false,
        badge: item.badge,
      })),
      {
        title: "navigation.agent_config",
        url: "/agent/config",
        icon: IconFileText,
        translateTitle: true,
      },
      {
        title: "navigation.agent_memory",
        url: "/agent/memory",
        icon: IconBrain,
        translateTitle: true,
      },
      {
        title: "navigation.skills",
        url: "/agent/skills",
        icon: IconSparkles,
        translateTitle: true,
      },
      {
        title: "navigation.tools",
        url: "/agent/tools",
        icon: IconTools,
        translateTitle: true,
      },
      {
        title: "navigation.cron",
        url: "/agent/cron",
        icon: IconClock,
        translateTitle: true,
      },
      ...servicesItems,
    ]
    return [
      {
        label: "navigation.longtrade",
        defaultOpen: true,
        items: [
          {
            title: "navigation.chat",
            url: "/",
            icon: IconMessageCircle,
            translateTitle: true,
          },
          {
            title: "navigation.paper_portfolio",
            url: "/portfolio",
            icon: IconChartPie,
            translateTitle: true,
          },
          {
            title: "navigation.try_trade",
            url: "/try-trade",
            icon: IconTrendingUp,
            translateTitle: true,
          },
        ],
      },
      {
        label: "navigation.advanced",
        defaultOpen: false,
        items: advancedItems,
      },
    ]
  }, [channelItems, sandboxStatus?.enabled])

  return (
    <Sidebar
      {...props}
      className="bg-background border-r-border/20 border-r pt-3"
    >
      <SidebarContent className="bg-background">
        {navGroups.map((group) => (
          <Collapsible
            key={group.label}
            defaultOpen={group.defaultOpen}
            className="group/collapsible mb-1"
          >
            <SidebarGroup className="px-2 py-0">
              <SidebarGroupLabel asChild>
                <CollapsibleTrigger className="hover:bg-muted/60 flex w-full cursor-pointer items-center justify-between rounded-md px-2 py-1.5 transition-colors">
                  <span>{t(group.label)}</span>
                  <IconChevronRight className="size-3.5 opacity-50 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                </CollapsibleTrigger>
              </SidebarGroupLabel>
              <CollapsibleContent>
                <SidebarGroupContent className="pt-1">
                  <SidebarMenu>
                    {group.items.map((item) => {
                      const isActive =
                        currentPath === item.url ||
                        (item.url !== "/" &&
                          currentPath.startsWith(`${item.url}/`))
                      return (
                        <SidebarMenuItem key={item.title}>
                          <SidebarMenuButton
                            asChild
                            isActive={isActive}
                            className={`h-9 px-3 ${isActive ? "bg-accent/80 text-foreground font-medium" : "text-muted-foreground hover:bg-muted/60"}`}
                          >
                            <Link to={item.url}>
                              <item.icon
                                className={`size-4 ${isActive ? "opacity-100" : "opacity-60"}`}
                              />
                              <span
                                className={
                                  isActive ? "opacity-100" : "opacity-80"
                                }
                              >
                                {item.translateTitle === false
                                  ? item.title
                                  : t(item.title)}
                              </span>
                              {item.badge && <AlertDot className="ml-auto" />}
                            </Link>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      )
                    })}
                    {group.isChannelsGroup && hasMoreChannels && (
                      <SidebarMenuItem key="channels-more-toggle">
                        <SidebarMenuButton
                          onClick={toggleShowAllChannels}
                          className="text-muted-foreground hover:bg-muted/60 h-9 px-3"
                        >
                          {showAllChannels ? (
                            <IconChevronsUp className="size-4 opacity-60" />
                          ) : (
                            <IconChevronsDown className="size-4 opacity-60" />
                          )}
                          <span className="opacity-80">
                            {showAllChannels
                              ? t("navigation.show_less_channels")
                              : t("navigation.show_more_channels")}
                          </span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    )}
                  </SidebarMenu>
                </SidebarGroupContent>
              </CollapsibleContent>
            </SidebarGroup>
          </Collapsible>
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}

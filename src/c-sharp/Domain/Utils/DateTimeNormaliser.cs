
using System;

namespace Archlens.Domain.Utils;

public static class DateTimeNormaliser
{
    public static DateTime NormaliseUTC(DateTime time)
    {
        var convertedDate = DateTime.SpecifyKind(time, DateTimeKind.Utc);
        return convertedDate.ToLocalTime();
    }
}
